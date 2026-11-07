# Changelog

Semua perubahan penting pada project ini dicatat di file ini.

Format mengikuti [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
dan versi mengikuti [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Scan repository git secara paralel dengan worker pool dan output deterministik.
- Deteksi changed, untracked, ahead, behind, no upstream, upstream gone, no remote, no commits, dan detached HEAD.
- Mode teks berwarna dengan kolom rata dan ringkasan yang informatif.
- Mode JSON (`-json`) dengan schema version, summary, dan field `attention`.
- Opsi `-hidden` untuk memindai folder tersembunyi.
- Opsi `-fetch` (opt-in) untuk ahead/behind yang lebih akurat.
- Opsi `-timeout` untuk membatasi durasi tiap perintah git.
- Opsi `-color auto|always|never`, plus dukungan `NO_COLOR`, `CLICOLOR`, `FORCE_COLOR`, dan `TERM=dumb`.
- Timeout, signal handling, dan exit code yang konsisten.
- Makefile, GitHub Actions CI, integration test, race test, dan benchmark.
- Logo, banner, terminal demo, dan dokumentasi lengkap.

### Security

- Default read-only: tidak fetch, tidak menulis index, dan tidak meninggalkan `index.lock`.
