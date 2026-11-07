# Skills used for this audit

## Installed

### `golang-security` (samber/cc-skills-golang)

- Source: <https://github.com/samber/cc-skills-golang> (MIT)
- Installed to: `~/.config/opencode/skills/golang-security/`
- Contents: `SKILL.md` + 12 markdown references (injection, filesystem, secrets,
  logging, threat modeling, checklist, …). No executable scripts.
- Why: Go-specific security review covering the exact categories this codebase
  touches (command execution, file paths, secrets, logging).

It becomes available to new sessions after OpenCode reloads the skills
directory (`/reload` or restart). Its checklist was applied manually during this
audit together with `staticcheck`, `gosec`, and `govulncheck`.

## Recommended, not installed

| Source | Why it is useful | Notes |
|---|---|---|
| <https://github.com/trailofbits/skills> | Specialist audit plugins: `static-analysis`, `variant-analysis`, `differential-review`, `insecure-defaults`, `supply-chain-risk-auditor` | Plugin marketplace, CC-BY-SA-4.0. Pick individual plugins. |
| <https://github.com/cloudflare/security-audit-skill> | Multi-agent repository audit with independent verification | MIT. Heavier; good for a full pre-release audit. |
| <https://github.com/mrevjd/claude-review-suite> | `review-go` bundles `go vet`, `staticcheck`, `gosec`, `errcheck`, `govulncheck` | Reports only, does not auto-fix. |
| <https://github.com/saisudhir14/claude-skills> | 9 Go skills, explicitly OpenCode-compatible (`go-security`, `go-code-review`, `go-project-layout`) | MIT. |
| <https://github.com/openai/skills> | `security-best-practices`, `security-threat-model` | Codex catalog; instructions are provider-agnostic. |
| <https://github.com/anthropics/skills> | Official example skills and format reference | Use as the source of truth for `SKILL.md` structure. |

## Install pattern

```bash
git clone --depth 1 https://github.com/<owner>/<repo> /tmp/skills
# Review SKILL.md and any scripts before copying.
mkdir -p ~/.config/opencode/skills/<skill-name>
cp -r /tmp/skills/<path-to-skill>/. ~/.config/opencode/skills/<skill-name>/
```

## Safety rules for third-party skills

1. Read `SKILL.md` and every script before installing; skills are executable
   instructions in your agent's context.
2. Prefer repositories with a license, an author with a track record, and no
   obfuscated shell.
3. Never install a skill that asks for credentials, tunnels, or broad `Bash(*)`
   permissions for a task that does not need them.
4. Pin a commit when reproducibility matters.
5. Scan skills with a tool such as `skill-scanner` or `SkillSpector` if you
   install from an unfamiliar source.
