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

target "dockerfile" {
  dockerfile = "Dockerfile"
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
