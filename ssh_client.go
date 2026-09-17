package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// neoMCPSSHUser logs into full neo-shell (DB session, /usr/bin sql/show/import/export).
// neoMCPSSHJshUser selects the raw jsh shellId so a "-C <script>" exec request runs
// the script once and exits, instead of neo-shell's "jsh" alias (/sbin/shell.js)
// which always starts an interactive REPL and ignores -C, hanging forever on a
// non-interactive SSH exec. Trade-off: the raw jsh engine has no DB session, no
// NEOSHELL_* login, and no /usr/bin commands - see neo-mcp/manual/jsh.md.
const (
	neoMCPSSHUser    = "neo-mcp"
	neoMCPSSHJshUser = "neo-mcp:jsh"
)

// SSHClient connects to the machbase-neo SSH (shell) service. Its address is
// never configured directly; it is discovered on demand via the
// "service.port.list" JSON-RPC API (Client.ServicePorts) so it always
// reflects the server's actual listener configuration.
type SSHClient struct {
	client *Client
	token  string
}

type JSHResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func NewSSHClient(client *Client, token string) *SSHClient {
	return &SSHClient{client: client, token: token}
}

func (c *SSHClient) RunNeoShellCommand(ctx context.Context, command string) (JSHResult, error) {
	return c.run(ctx, neoMCPSSHUser, command)
}

func (c *SSHClient) RunJSH(ctx context.Context, script string) (JSHResult, error) {
	// shell.Cmd for the "jsh" shellId is already the jsh engine itself, so the exec
	// command only carries "-C <script>", not a "jsh" prefix.
	return c.run(ctx, neoMCPSSHJshUser, "-C "+shellQuote(script))
}

func (c *SSHClient) RunJSHFile(ctx context.Context, publicPath string) (JSHResult, error) {
	path, err := normalizeJSHFilePath(publicPath)
	if err != nil {
		return JSHResult{}, err
	}
	if !strings.HasSuffix(strings.ToLower(path), ".js") {
		return JSHResult{}, fmt.Errorf("JSH file must have .js extension: %q", publicPath)
	}
	return c.RunJSH(ctx, "require("+strconv.Quote(path)+")")
}

// resolveAddress discovers the SSH (shell) service listener address by
// querying the machbase-neo "service.port.list" JSON-RPC API.
func (c *SSHClient) resolveAddress(ctx context.Context) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("machbase-neo API client is required to discover the SSH server address")
	}
	ports, err := c.client.ServicePorts(ctx, "shell")
	if err != nil {
		return "", fmt.Errorf("failed to discover SSH server address: %w", err)
	}
	for _, port := range ports {
		if addr, ok := strings.CutPrefix(port.Address, "tcp://"); ok && addr != "" {
			return addr, nil
		}
	}
	return "", fmt.Errorf("machbase-neo did not report an SSH (shell) service address")
}

func (c *SSHClient) run(ctx context.Context, user string, command string) (JSHResult, error) {
	if c.token == "" {
		return JSHResult{}, fmt.Errorf("API token is required for SSH authentication")
	}
	address, err := c.resolveAddress(ctx)
	if err != nil {
		return JSHResult{}, err
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(c.token)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return JSHResult{}, fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return JSHResult{}, fmt.Errorf("SSH session failed: %w", err)
	}
	defer session.Close()

	stdout, stderr, exitCode, runErr := runSSHCommand(ctx, session, command)
	if runErr != nil {
		return JSHResult{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}, runErr
	}
	return JSHResult{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}, nil
}

func runSSHCommand(ctx context.Context, session *ssh.Session, command string) (string, string, int, error) {
	stdout, stderr, err := captureSessionOutput(session)
	if err != nil {
		return "", "", -1, err
	}
	if err := session.Start(command); err != nil {
		return "", "", -1, fmt.Errorf("SSH command start failed: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			if exitError, ok := err.(*ssh.ExitError); ok {
				return stdout.String(), stderr.String(), exitError.ExitStatus(), nil
			}
			return stdout.String(), stderr.String(), -1, err
		}
		return stdout.String(), stderr.String(), 0, nil
	case <-ctx.Done():
		_ = session.Close()
		return stdout.String(), stderr.String(), -1, ctx.Err()
	}
}

func captureSessionOutput(session *ssh.Session) (*strings.Builder, *strings.Builder, error) {
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	session.Stdout = stdout
	session.Stderr = stderr
	return stdout, stderr, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func sshAddress(host string, port int) string {
	return net.JoinHostPort(host, fmt.Sprint(port))
}
