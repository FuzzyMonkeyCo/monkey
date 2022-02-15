## Groups

group "default" {
  targets = [
    "binaries",
  ]
}

group "ci-checks" {
  targets = [
    "ci-check--lint",
    "ci-check--mod",
    "ci-check--test",
  ]
}

## Targets

target "dockerfile" {
  dockerfile = "Dockerfile"
}

target "binaries" {
  inherits = ["dockerfile"]
  target = "binaries"
  output = ["."]
  cache-from = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:binaries"]
  # cache-to = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:binaries,mode=max"]
}

target "goreleaser-dist" {
  inherits = ["dockerfile"]
  target = "goreleaser-dist"
  output = ["./dist"]
  cache-from = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:goreleaser-dist"]
  # cache-to only supported in CI? (must not be using docker driver)
  # cache-to = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:goreleaser-dist,mode=max"]
}

target "ci-check--lint" {
  inherits = ["dockerfile"]
  target = "ci-check--lint"
  cache-from = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:ci-check--lint"]
  # cache-to = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:ci-check--lint,mode=max"]
}

target "ci-check--mod" {
  inherits = ["dockerfile"]
  target = "ci-check--mod"
  cache-from = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:ci-check--mod"]
  # cache-to = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:ci-check--mod,mode=max"]
}

target "ci-check--test" {
  inherits = ["dockerfile"]
  target = "ci-check--test"
  cache-from = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:ci-check--test"]
  # cache-to = ["type=registry,ref=ghcr.io/fuzzymonkeyco/monkey:ci-check--test,mode=max"]
}
