// Requires Go toolchain: atlas migrate diff --env gorm calls go run ./cmd/loader/main.go locally.
// CI lint uses atlas-community without --env, so Go is not required in CI.
data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/loader/main.go",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/dev?search_path=public"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
