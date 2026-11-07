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

1. runs `go test -race ./...`;
2. builds `dist/` with `scripts/build.sh all`, injecting version, commit, and
   build date;
3. verifies `dist/SHA256SUMS`;
4. creates a GitHub Release with the artifacts and generated notes.

## 4. Verify a release

```bash
sha256sum -c SHA256SUMS
./dist/dc2-linux-amd64   # /dev info shows the tag and commit
```

Check the release assets: linux amd64/arm64, windows amd64, darwin arm64, and
`SHA256SUMS`.

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
