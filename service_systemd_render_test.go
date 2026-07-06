//go:build linux

package service

import (
	"strconv"
	"testing"
)

// systemdInputs collects the systemd-specific values Install passes to the
// template, so the render can be exercised without touching the filesystem.
type systemdInputs struct {
	c                 *Config
	path              string
	reloadSignal      string
	pidFile           string
	limitNOFILE       int
	restart           string
	successExitStatus string
	logOutput         bool
	hasOutput         bool
	logDir            string
}

// renderSystemd mirrors the data map built in (*systemd).Install and renders
// the built-in template through the engine.
func renderSystemd(in systemdInputs) (string, error) {
	c := in.c
	outputFileSupport := ""
	if in.logOutput && in.hasOutput {
		outputFileSupport = "yes"
	}
	limitNOFILE := ""
	if in.limitNOFILE > -1 {
		limitNOFILE = strconv.Itoa(in.limitNOFILE)
	}
	data := map[string]any{
		"Description":       c.Description,
		"Path":              in.path,
		"Name":              c.Name,
		"Dependencies":      c.Dependencies,
		"Arguments":         c.Arguments,
		"ChRoot":            c.ChRoot,
		"WorkingDirectory":  c.WorkingDirectory,
		"UserName":          c.UserName,
		"ReloadSignal":      in.reloadSignal,
		"PIDFile":           in.pidFile,
		"LogDirectory":      in.logDir,
		"OutputFileSupport": outputFileSupport,
		"LimitNOFILE":       limitNOFILE,
		"Restart":           in.restart,
		"SuccessExitStatus": in.successExitStatus,
		"EnvVars":           envVars(c.EnvVars, func(k, v string) string { return "Environment=" + k + "=" + v }),
	}
	return renderTemplate(systemdScript, data, tfs)
}

const systemdGoldenMinimal = `[Unit]
Description=A test
ConditionFileIsExecutable=/usr/bin/svc\x20app

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart=/usr/bin/svc\x20app
ExecReload=/bin/kill -SIGUSR1 "$MAINPID"
PIDFile="/var/run/svc.pid"
StandardOutput=file:/var/log/svc.out
StandardError=file:/var/log/svc.err
LimitNOFILE=4096
Restart=always
SuccessExitStatus=1 2
RestartSec=120
EnvironmentFile=-/etc/sysconfig/svc

[Install]
WantedBy=multi-user.target
`

const systemdGoldenFull = `[Unit]
Description=A test service
ConditionFileIsExecutable=/usr/bin/svc\x20app
After=network.target
Requires=foo.service

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart=/usr/bin/svc\x20app "-a" "with space"
RootDirectory="/jail"
WorkingDirectory=/opt/my\x20app
User=bob
ExecReload=/bin/kill -SIGUSR1 "$MAINPID"
PIDFile="/var/run/svc.pid"
StandardOutput=file:/var/log/svc.out
StandardError=file:/var/log/svc.err
LimitNOFILE=4096
Restart=always
SuccessExitStatus=1 2
RestartSec=120
EnvironmentFile=-/etc/sysconfig/svc

Environment=A=1
Environment=B=2
[Install]
WantedBy=multi-user.target
`

func TestSystemdRender(t *testing.T) {
	base := systemdInputs{
		path: "/usr/bin/svc app", reloadSignal: "SIGUSR1", pidFile: "/var/run/svc.pid",
		limitNOFILE: 4096, restart: "always", successExitStatus: "1 2",
		logOutput: true, hasOutput: true, logDir: "/var/log",
	}
	cases := []struct {
		name string
		c    *Config
		want string
	}{
		{"minimal", &Config{Name: "svc", Description: "A test"}, systemdGoldenMinimal},
		{"full", &Config{
			Name:             "svc",
			Description:      "A test service",
			UserName:         "bob",
			Arguments:        []string{"-a", "with space"},
			Dependencies:     []string{"After=network.target", "Requires=foo.service"},
			WorkingDirectory: "/opt/my app",
			ChRoot:           "/jail",
			EnvVars:          map[string]string{"B": "2", "A": "1"},
		}, systemdGoldenFull},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			in.c = tc.c
			got, err := renderSystemd(in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("output mismatch\n--- got ---\n%q\n--- want ---\n%q", got, tc.want)
			}
		})
	}
}
