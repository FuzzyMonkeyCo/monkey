package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadBadVersions(t *testing.T) {
	for name, config := range map[string][]byte{
		"empty":                     []byte(``),
		"typo in key":               []byte(`verion: 4`),
		"unsupported version":       []byte(`version: 42`),
		"bad minimum":               []byte(`version: 0`),
		"float":                     []byte(`version: 4.2`),
		"string a la docker config": []byte(`version: '4'`),
		"missing value":             []byte(`version: `),
		"typo in value":             []byte(`version:\n1`),
		"duplicate key":             []byte(`{version: 1, port: 80, port: 443}`),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := parseCfg(config, false)
			require.Error(t, err)
			require.Equal(t, (*UserCfg)(nil), cfg)
		})
	}
}

func TestV1ReadErrors(t *testing.T) {
	for name, config := range map[string][]byte{
		"duplicate key": []byte(`
version: 1
spec:
  kind: OpenAPIv3
  file: some_file.json
spec: blbl
`),
		"unexpected key": []byte(`
version: 1
spec:
  kind: OpenAPIv3
  file: some_file.json
blabla: blbl
`),
		"more than one issue at once": []byte(`
version: 1
spec: yes
spec:
  kind: OpenAPIv3
  file: some_file.json
blabla: blbl
`),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := parseCfg(config, false)
			require.Error(t, err)
			require.Equal(t, (*UserCfg)(nil), cfg)
		})
	}
}

func TestV1ReadAllSet(t *testing.T) {
	config := []byte(`
version: 1
spec:
  host: https://app.vcap.me:{{ env "MY_PORT" }}
  file: ./spec.yml
  kind: OpenAPIv3
  authorization: Bearer xyz
start:
- make service-start
reset:
- make service-restart
stop:
- make service-kill
`)

	cfg, err := parseCfg(config, false)
	require.NoError(t, err)
	require.Equal(t, "app.vcap.me", cfg.Runtime.Host)
	require.Equal(t, "8000", cfg.Runtime.Port)
	require.Equal(t, "./spec.yml", cfg.File)
	require.Equal(t, UserCfg_OpenAPIv3, cfg.Kind)
	require.Equal(t, "OpenAPIv3", cfg.Kind.String())
	require.Equal(t, "Bearer xyz", *addHeaderAuthorization)
	require.Equal(t, []string{"make service-start"}, cfg.Exec.Start)
	require.Equal(t, []string{"make service-restart"}, cfg.Exec.Reset_)
	require.Equal(t, []string{"make service-kill"}, cfg.Exec.Stop)
}

func TestV1ReadDefaults(t *testing.T) {
	for name, config := range map[string][]byte{
		"bare minimum": []byte(`
version: 1
spec:
  kind: OpenAPIv3
`),
		"version & host": []byte(`
version: 1
spec:
  kind: OpenAPIv3
  host: localhost
`),
		"version & port": []byte(`
version: 1
spec:
  kind: OpenAPIv3
  port: 3000
`),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := parseCfg(config, false)
			require.NoError(t, err)
			require.Equal(t, "localhost", cfg.Runtime.Host)
			require.Equal(t, "3000", cfg.Runtime.Port)
		})
	}
}
