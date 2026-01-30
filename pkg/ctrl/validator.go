// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

package ctrl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Validator struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
}

func NewValidator(ctx context.Context, credsPath, caPath, ref, tag string) (*Validator, error) {
	v := &Validator{}

	if ref == "" && tag == "" {
		slog.Info("Skipping Dataplane validator as it is not configured")

		return nil, nil //nolint:nilnil
	}

	slog.Info("Loading dataplane validator", "version", tag)

	slog.Debug("Downloading dataplane validator", "ref", ref)

	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStore(credsPath, storeOpts)
	if err != nil {
		return nil, fmt.Errorf("creating docker credential store for %s: %w", credsPath, err)
	}

	ca, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("reading CA cert %s: %w", caPath, err)
	}

	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("appending CA cert to rootCAs: %w", err)
	}

	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
		RootCAs:            rootCAs,
	}

	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("creating oras remote repo %s: %w", ref, err)
	}

	repo.Client = &auth.Client{
		Client: &http.Client{
			Transport: retry.NewTransport(baseTransport),
		},
		Cache:      auth.DefaultCache,
		Credential: credentials.Credential(credStore),
	}

	tmp, err := os.MkdirTemp("", "download-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	fs, err := file.New(tmp)
	if err != nil {
		return nil, fmt.Errorf("creating oras file store %s: %w", tmp, err)
	}
	defer fs.Close()

	_, err = oras.Copy(ctx, repo, tag, fs, "", oras.CopyOptions{
		CopyGraphOptions: oras.CopyGraphOptions{
			Concurrency: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("downloading files from %s:%s: %w", ref, tag, err)
	}

	wasmBytes, err := os.ReadFile(filepath.Join(tmp, "dataplane-validator"))
	if err != nil {
		return nil, fmt.Errorf("reading WASM file: %w", err)
	}

	slog.Debug("Setting up WASM runtime")

	v.runtime = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, v.runtime)

	slog.Debug("Compiling dataplane validator")

	v.compiled, err = v.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compiling WASM module: %w", err)
	}

	return v, nil
}

func (v *Validator) Close(ctx context.Context) {
	if err := v.compiled.Close(ctx); err != nil {
		slog.Warn("Error closing compiled validator module", "err", err.Error())
	}
	if err := v.runtime.Close(ctx); err != nil {
		slog.Warn("Error closing validator runtime", "err", err.Error())
	}
}
