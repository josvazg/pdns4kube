//go:build mage

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	// pdnsContainerPort is the PowerDNS API port inside the cluster.
	pdnsContainerPort = "8081"
	// pdnsHostPort is the local forwarded port. It deliberately differs
	// from the container port to avoid colliding with the operator's
	// default health probe bind address (:8081) during `mage run`.
	pdnsHostPort = "18081"
)

// GenCRD runs openapi2crd to generate the Kubernetes Custom Resource Definitions.
func GenCRD() error {
	fmt.Println("Generating CRDs from OpenAPI spec...")
	return sh.RunV(
		"go", "tool", "openapi2crd",
		"--config", "openapi2crd.yaml",
		"--output", "config/crd/bases/pdns.example.io.yaml",
		"--force",
	)
}

// GenGo runs crd2go and controller-gen to generate the Go types from the CRDs
func GenGo() error {
	fmt.Println("Generating Go types from CRDs...")
	err := sh.RunV("go", "tool", "crd2go",
		"--input", "config/crd/bases/pdns.example.io.yaml",
		"--output", "v1")
	if err != nil {
		return fmt.Errorf("CRD2Go failed: %w", err)
	}
	return sh.RunV("go", "tool", "controller-gen", "object", "paths=./v1/...")
}

// KindUp creates the kind cluster named pdns4kube if it is not already running.
func KindUp() error {
	out, err := sh.Output("kind", "get", "clusters")
	if err != nil {
		return fmt.Errorf("kind get clusters failed: %w", err)
	}
	if strings.Contains(out, "pdns4kube") {
		fmt.Println("cluster pdns4kube already running")
		return nil
	}
	return sh.RunV("kind", "create", "cluster", "--name", "pdns4kube")
}

// KindDown deletes the kind cluster named pdns4kube.
func KindDown() error {
	return sh.RunV("kind", "delete", "cluster", "--name", "pdns4kube")
}

// InstallCRD applies the generated CRDs to the current cluster.
func InstallCRD() error {
	fmt.Println("Installing CRDs...")
	return sh.RunV("kubectl", "apply", "-f", "config/crd/bases/pdns.example.io.yaml")
}

// PDNSUp ensures the kind cluster is running, then applies the dev PowerDNS
// manifest and waits for the deployment to roll out.
func PDNSUp() error {
	mg.SerialDeps(KindUp)
	fmt.Println("Applying PDNS dev manifest...")
	if err := sh.RunV("kubectl", "apply", "-f", "config/dev/pdns.yaml"); err != nil {
		return err
	}
	return sh.RunV("kubectl", "rollout", "status", "deployment/pdns-dev", "--timeout=120s")
}

// PDNSDown deletes the dev PowerDNS resources, ignoring missing ones.
func PDNSDown() error {
	return sh.RunV("kubectl", "delete", "-f", "config/dev/pdns.yaml", "--ignore-not-found=true")
}

// PDNSForward ensures PDNSUp is applied, then port-forwards the service to
// http://127.0.0.1:18081 (auth header: X-API-Key: pdns4kube-dev-key).
// This blocks until the port-forward is interrupted.
func PDNSForward() error {
	mg.SerialDeps(PDNSUp)

	pf, err := startPortForward()
	if err != nil {
		return err
	}
	defer pf.stop()

	fmt.Printf("API endpoint: http://127.0.0.1:%s (header: X-API-Key: pdns4kube-dev-key)\n", pdnsHostPort)

	// Block until the port-forward exits on its own or the user interrupts,
	// so the deferred pf.stop() cleanup always runs on Ctrl+C/SIGTERM.
	sig := notifyInterrupt()
	defer signal.Stop(sig)
	select {
	case <-pf.done:
		return pf.err
	case <-sig:
		fmt.Println("Interrupt received, stopping port-forward...")
		return nil
	}
}

// Run runs the operator locally, outside of the cluster, starting the kind
// cluster first if needed, installing the CRDs, port-forwarding the dev PDNS
// service, and then running the operator. The port-forward is stopped when the
// operator exits. Use `mage pdnsDown` for explicit cleanup of PDNS resources.
func Run() error {
	mg.SerialDeps(KindUp, InstallCRD, PDNSUp)

	pf, err := startPortForward()
	if err != nil {
		return err
	}
	defer pf.stop()

	fmt.Printf("API endpoint: http://127.0.0.1:%s (header: X-API-Key: pdns4kube-dev-key)\n", pdnsHostPort)
	fmt.Println("Running operator locally (press Ctrl+C to stop)...")

	// Run the operator; relay its output.
	operatorCmd := exec.Command("go", "run", "./cmd")
	// Dev-only PowerDNS endpoint values, scoped to this local Mage run.
	operatorCmd.Env = setEnv(os.Environ(),
		"PDNS_API_URL", "http://127.0.0.1:"+pdnsHostPort,
		"PDNS_API_KEY", "pdns4kube-dev-key")
	operatorCmd.Stdout = os.Stdout
	operatorCmd.Stderr = os.Stderr
	operatorCmd.Stdin = os.Stdin
	if err := operatorCmd.Start(); err != nil {
		return fmt.Errorf("starting operator: %w", err)
	}

	sig := notifyInterrupt()
	defer signal.Stop(sig)
	return waitForOperator(operatorCmd, sig)
}

// setEnv returns env with each key=value pair set, replacing any existing
// value for the key instead of duplicating it.
func setEnv(env []string, pairs ...string) []string {
	out := make([]string, 0, len(env)+len(pairs))
	set := make(map[string]bool, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		set[pairs[i]] = true
	}
	for _, e := range env {
		k, _, ok := strings.Cut(e, "=")
		if ok && set[k] {
			continue
		}
		out = append(out, e)
	}
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, pairs[i]+"="+pairs[i+1])
	}
	return out
}

// waitForOperator waits for the operator to exit or for the user to interrupt.
// On interrupt, it gives the operator time to shut down before killing it.
func waitForOperator(operatorCmd *exec.Cmd, sig <-chan os.Signal) error {
	opErr := make(chan error, 1)
	go func() { opErr <- operatorCmd.Wait() }()

	select {
	case err := <-opErr:
		if err != nil {
			return fmt.Errorf("operator exited with error: %w", err)
		}
		return nil
	case <-sig:
		fmt.Println("Interrupt received, shutting down...")
		// The operator (a child in the same process group) also received
		// the interrupt; give it time to exit, then kill it.
		select {
		case err := <-opErr:
			fmt.Printf("operator exited with: %v\n", err)
		case <-time.After(10 * time.Second):
			_ = operatorCmd.Process.Kill()
			<-opErr
		}
		return nil
	}
}

// notifyInterrupt installs a temporary handler for SIGINT/SIGTERM and returns
// a channel that receives those signals. Call signal.Stop on the channel when
// done to restore default signal behavior.
func notifyInterrupt() chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	return sig
}

// portForward manages a `kubectl port-forward service/pdns-dev 18081:8081`
// child process: it relays output, waits for 127.0.0.1:18081 readiness, and
// stops/reaps the process on cleanup.
type portForward struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error
}

// startPortForward starts the kubectl port-forward child process and waits
// (bounded) for 127.0.0.1:18081 to become reachable.
func startPortForward() (*portForward, error) {
	cmd := exec.Command("kubectl", "port-forward", "service/pdns-dev", pdnsHostPort+":"+pdnsContainerPort)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting kubectl port-forward: %w", err)
	}

	pf := &portForward{cmd: cmd, done: make(chan struct{})}
	go func() {
		pf.err = cmd.Wait()
		close(pf.done)
	}()

	fmt.Printf("Port-forward running (kubectl port-forward service/pdns-dev %s:%s)\n", pdnsHostPort, pdnsContainerPort)
	if err := pf.waitReady(30 * time.Second); err != nil {
		pf.stop()
		return nil, err
	}
	return pf, nil
}

// waitReady polls 127.0.0.1:18081 until it accepts connections, the
// port-forward process exits early, or the timeout elapses.
func (pf *portForward) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-pf.done:
			return fmt.Errorf("kubectl port-forward exited early with: %v", pf.err)
		default:
		}
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+pdnsHostPort, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for 127.0.0.1:%s to become ready: %w", timeout, pdnsHostPort, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// stop interrupts the port-forward process, waits up to 5 seconds for it to
// exit, then kills it and reaps it.
func (pf *portForward) stop() {
	_ = pf.cmd.Process.Signal(os.Interrupt)
	select {
	case <-pf.done:
	case <-time.After(5 * time.Second):
		_ = pf.cmd.Process.Kill()
		<-pf.done
	}
}
