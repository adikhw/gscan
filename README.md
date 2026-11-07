<div align="center">

<img src="https://go.dev/images/go-logo-blue.svg" alt="Go" width="110">

# gscan

**Scan repo git lokal, temukan pekerjaan yang lupa di-commit atau di-push.**

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen)](go.mod)

</div>

`gscan` memindai semua repository git di bawah sebuah folder (default: home
directory) dan langsung menunjukkan mana yang perlu perhatian: file belum
di-commit, file untracked, commit belum di-push, branch tanpa upstream, repo
tanpa remote, detached HEAD, sampai repo yang rusak.

```text
$ gscan
~/projects/capri        main             2 changed, 1 untracked, ahead 2
~/projects/yap          main             no remote
~/projects/myReminder   feature/login    1 untracked, no upstream

Scanned 3 repositories in 42ms - 3 need attention (1 dirty, 1 unpushed)
```

## Fitur

- Scan home directory secara default, atau folder tertentu.
- Paralel dengan worker pool, hasil selalu diurutkan berdasarkan path.
- Read-only secara default: tidak fetch, tidak menulis index.
- Aman untuk folder besar: dependency, cache, dan build artifact dilewati.
- Tidak mengikuti symlink dan tidak menghentikan scan karena satu repo error.
- Output teks berwarna dan rapi, plus mode `-json` untuk scripting.
- Zero third-party dependency, hanya standard library Go.

## Kebutuhan

| Dependency | Versi | Dipakai untuk |
|---|---|---|
| [Go](https://go.dev) | 1.21+ | Build dari source |
| [git](https://git-scm.com) | 2.20+ | Runtime |
| Terminal ANSI | Opsional | Warna output |

## Instalasi

```bash
# Paling cepat
go install github.com/adikhw/gscan@latest

# Dari source
git clone https://github.com/adikhw/gscan.git
cd gscan
make build
./gscan -version

# Install ke ~/.local/bin
make install
```

Pastikan binari Go ada di `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin:$HOME/.local/bin"
```

## Penggunaan

```text
gscan [options] [path]

Arguments:
  path                 folder yang dipindai (default: home directory)

Options:
  -all                 tampilkan juga repo yang bersih
  -j N                 jumlah worker paralel (default: jumlah CPU)
  -hidden              ikut pindai folder tersembunyi
  -json                output JSON
  -color MODE          auto, always, atau never (default: auto)
  -timeout DURATION    timeout per perintah git (default: 10s)
  -fetch               fetch remote dulu (butuh jaringan, menulis .git)
  -version             tampilkan versi
  -h, --help           tampilkan bantuan
```

Flag boleh ditaruh sebelum atau sesudah path, jadi `gscan -all ~/src` dan
`gscan ~/src -all` sama-sama valid.

### Contoh

```bash
gscan                       # scan seluruh home
gscan ~/projects            # scan folder tertentu
gscan .                     # scan folder saat ini
gscan -all                  # tampilkan repo bersih juga
gscan -hidden ~/.config     # ikut folder tersembunyi
gscan -j 2 ~/work           # batasi worker
gscan -json > report.json   # output untuk scripting
gscan -fetch ~/work         # ahead/behind lebih akurat
```

Exit code: `0` scan selesai, `1` gagal runtime, `2` usage salah,
`130` dihentikan dengan Ctrl+C.

## Arti status

| Status | Arti |
|---|---|
| `N changed` | Ada `N` file tracked yang berubah |
| `N untracked` | Ada `N` file baru yang belum di-`git add` |
| `ahead N` | Ada `N` commit lokal yang belum di-push |
| `behind N` | Remote punya `N` commit yang belum ada di lokal |
| `no upstream` | Branch tidak punya upstream, tapi remote ada |
| `upstream gone` | Upstream branch sudah dihapus di remote |
| `no remote` | Repo tidak punya remote |
| `no commits` | Repo baru, belum ada commit |
| `detached` | HEAD sedang detached |
| `clean` | Tidak ada perubahan dan sudah sinkron |
| `error: ...` | Repo rusak atau git gagal dijalankan |

`behind` saja tidak dianggap “perlu perhatian” karena tidak ada pekerjaan
lokal yang berisiko hilang, tapi tetap ditampilkan di ringkasan.

## Contoh JSON

`gscan -json` selalu mengeluarkan semua repo yang ditemukan.

```json
{
  "schema": "1",
  "root": "/home/user",
  "workers": 8,
  "summary": {
    "total": 1,
    "attention": 1,
    "clean": 0,
    "dirty": 1,
    "unpushed": 0,
    "behind": 0,
    "errors": 0
  },
  "repositories": [
    {
      "path": "/home/user/projects/capri",
      "branch": "main",
      "upstream": "origin/main",
      "has_upstream": true,
      "changed": 2,
      "untracked": 1,
      "attention": true,
      "status": ["2 changed", "1 untracked"]
    }
  ]
}
```

## Development

```bash
make build     # build ./gscan
make test      # go test ./...
make race      # go test -race ./...
make vet       # go vet ./...
make check     # gofmt check + vet + race test
make bench     # benchmark
make clean     # hapus binary
```

Test mencakup parser status, heuristik “perlu perhatian”, worker pool, output
teks/JSON, timeout, read-only behavior, dan integrasi dengan repo git sementara.

## License

[MIT](LICENSE) © 2026 adikhw
