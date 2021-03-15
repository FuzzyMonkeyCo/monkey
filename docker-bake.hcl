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
  cache-from = ["type=registry,ref=fenollp/monkey:cache"]
  cache-to = ["type=registry,ref=fenollp/monkey:cache,mode=max"]
}

target "binaries" {
  inherits = ["dockerfile"]
  target = "binaries"
  output = ["."]
}

target "goreleaser-dist" {
  inherits = ["dockerfile"]
  target = "goreleaser-dist"
  output = ["./dist"]
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
