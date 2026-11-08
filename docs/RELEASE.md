# Release

## 1. Versioning

Tags are `vMAJOR.MINOR.PATCH` (for example `v0.3.0`). The tag becomes the
`version.Version` baked into the binary and shown by `/dev info`.

## 2. Pre-release checklist

- [ ] Working tree clean (`git status`).
- [ ] `make check` passes locally (fmt, vet, tests).
- [ ] CI on `main` is green, including the Windows job.
- [ ] `docs/` reflects any changed command or config.
- [ ] No token or `.env` in the tree (`git grep` for the token prefix).
- [ ] Lab smoke test passes (`lab/run.sh`).

## 3. Cut a release

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

Pushing the tag runs `.github/workflows/release.yml`, which:

1. runs `staticcheck`, `govulncheck`, and `gosec` (the release is blocked if any
   fail);
2. runs `go test -race ./...`;
3. builds `dist/` with `scripts/build.sh all`, injecting version, commit, and
   build date;
4. verifies `dist/SHA256SUMS`;
5. creates a GitHub Release with the artifacts and generated notes.

## 3.1 SBOM and Windows signing

Every release includes `dist/sbom.cdx.json` (CycloneDX, generated with
`cyclonedx-gomod v1.12.0` from the locked module graph).

Windows signing is optional and secret-driven. Add these repository secrets:

| Secret | Value |
|---|---|
| `WINDOWS_SIGNING_PFX_BASE64` | base64 of a `.pfx` code-signing certificate |
| `WINDOWS_SIGNING_PFX_PASSWORD` | the `.pfx` password |

When `WINDOWS_SIGNING_PFX_BASE64` is set, CI installs `osslsigncode`, signs
`dc2-windows-amd64.exe`, and regenerates `SHA256SUMS`. Without the secret the
workflow skips signing and publishes the unsigned binary.

Verify a signed build:

```powershell
Get-AuthenticodeSignature .\dc2-windows-amd64.exe
```

Every release also gets a **build provenance attestation** for the binaries and
SBOM. Verify one:

```bash
gh attestation verify dc2-linux-amd64 --repo MarwanKhatib/D-C2
```

## 4. Verify a release

```bash
sha256sum -c SHA256SUMS
./dist/dc2-linux-amd64   # /dev info shows the tag and commit
```

Check the release assets: linux amd64/arm64, windows amd64, darwin arm64,
`sbom.cdx.json`, and `SHA256SUMS`.

## 5. Rollback

- Delete the GitHub release and tag, or publish a new patch tag.
- Note that clients already running an older build keep working until the bot
  token is rotated or the guild changes.
- If a release contained a credential (it should not), rotate the token and
  rebuild.

## 6. Artifact policy

- Do not embed `.env` (`-tags envembed`) in release artifacts.
- Do not attach engagement-specific builds to a public release.
- Keep the release page free of target names, operator IDs, or guild IDs.
