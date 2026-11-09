package mactool

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rm4n0s/errors"
)

// SignOptions are what Sign is asked for.
//
// Flutter: no counterpart
type SignOptions struct {
	// Hardened turns the hardened runtime on (`--options runtime`) with the entitlements of Entitlements(), which a bundle needs to be
	// notarized; the owner does the notarization (docs/GUIDE.md), the tool only prepares for it.
	Hardened bool
	// Identity is the signing identity; "-" (ad hoc) when empty. A Developer ID is the owner's.
	Identity string
	// Tool is the codesign program; /usr/bin/codesign when empty.
	Tool string
}

func (o SignOptions) tool() string {
	if o.Tool != "" {
		return o.Tool
	}
	return "/usr/bin/codesign"
}

// Sign signs the bundle with codesign: a signature on the bundle binds its Info.plist and resources to the executable, which
// Gatekeeper and the system's permission prompts read (an ad-hoc signature has no identity, so Gatekeeper still warns about an app
// that was downloaded). It fails with MacOSSignFailed, with codesign's output, when codesign cannot be run or says no.
//
// Flutter: no counterpart
func Sign(ctx context.Context, app string, o SignOptions) error {
	identity := o.Identity
	if identity == "" {
		identity = "-"
	}
	args := []string{"--force", "--sign", identity, "--timestamp=none"}
	if o.Hardened {
		dir, err := os.MkdirTemp("", "glutter-macos-sign-")
		if err != nil {
			return errors.NewErr(tagMacOSSignFailed, err).SetMetadata("app", app)
		}
		defer func() { _ = os.RemoveAll(dir) }() // best effort: a temporary directory
		ent := filepath.Join(dir, "entitlements.plist")
		if err := os.WriteFile(ent, Entitlements(), 0o600); err != nil {
			return errors.NewErr(tagMacOSSignFailed, err).SetMetadata("app", app)
		}
		args = append(args, "--options", "runtime", "--entitlements", ent)
	}
	args = append(args, app)
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, o.tool(), args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return errors.NewErr(tagMacOSSignFailed, err).SetMetadata("app", app, "output", out.String())
	}
	return nil
}

// Verify checks the signature of a bundle with `codesign --verify --strict`. It fails with MacOSSignFailed when the bundle is not
// signed, or was changed after it was.
//
// Flutter: no counterpart
func Verify(ctx context.Context, app string, o SignOptions) error {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, o.tool(), "--verify", "--strict", "--verbose=2", app)
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return errors.NewErr(tagMacOSSignFailed, err).SetMetadata("app", app, "output", out.String())
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSSignFailed = TagMacOSSignFailed

	// TagMacOSSignFailed is the tag of the error for a bundle that codesign cannot sign or verify.
	//
	// Flutter: no counterpart
	TagMacOSSignFailed = "MacOSSignFailed"
)
