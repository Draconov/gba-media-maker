## Summary

Describe the change and the user-visible behavior it affects.

## Affected areas

- [ ] Desktop app / local web UI
- [ ] Browser edition
- [ ] Video conversion/player
- [ ] Audio conversion/player
- [ ] Image/GIF handling
- [ ] Menu/theme/title-card system
- [ ] GBA runtime / `player_stub.bin`
- [ ] Project/ROM format or output naming
- [ ] Documentation only

## Verification

- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] GBA player rebuilt when `player/` changed
- [ ] `cd website && npm test` when website behavior changed
- [ ] Website production build checked when relevant
- [ ] Video compatibility checked when relevant
- [ ] Audio/image/GIF path checked when relevant
- [ ] ROM/project-format compatibility impact documented
- [ ] `CHANGELOG.md` / permanent docs updated for user-visible release changes
