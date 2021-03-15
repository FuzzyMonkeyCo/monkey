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
  cache-from = ["type=registry,ref=fenollp/monkey:ci-checks"]
  cache-to = ["type=registry,ref=fenollp/monkey:ci-checks,mode=max"]
}

## Targets

target "dockerfile" {
  dockerfile = "Dockerfile"
}

target "binaries" {
  inherits = ["dockerfile"]
  target = "binaries"
  output = ["."]
  cache-from = ["type=registry,ref=fenollp/monkey:binaries"]
  cache-to = ["type=registry,ref=fenollp/monkey:binaries,mode=max"]
}

target "goreleaser-dist" {
  inherits = ["dockerfile"]
  target = "goreleaser-dist"
  output = ["./dist"]
  cache-from = ["type=registry,ref=fenollp/monkey:goreleaser-dist"]
  cache-to = ["type=registry,ref=fenollp/monkey:goreleaser-dist,mode=max"]
}

target "ci-check--lint" {
  inherits = ["dockerfile"]
  target = "ci-check--lint"
}

target "ci-check--mod" {
  inherits = ["dockerfile"]
  target = "ci-check--mod"
}

target "ci-check--test" {
  inherits = ["dockerfile"]
  target = "ci-check--test"
}
