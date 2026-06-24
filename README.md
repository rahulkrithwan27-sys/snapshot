# snapshot

Save and restore your Ubuntu setup. Capture the apps you installed and your
config files, then bring them all back on a fresh machine — and choose exactly
what to restore with an interactive checklist.

## What it does

Reinstalling Ubuntu means losing hours rebuilding your environment. `snapshot`
fixes that. It captures:

- **apt packages** you installed (filtered to your own apps, not the base system)
- **snap packages**
- key **dotfiles** (`.bashrc`, `.gitconfig`, `.vimrc`, `.profile`)

On a new machine, `restore` shows everything in a checklist so you pick what you
actually want back.

## Install

Download the binary, or build from source (requires Go 1.22+):

```
git clone https://github.com/yourname/snapshot.git
cd snapshot
go build -o snapshot .
```

## Usage

```
./snapshot save              # capture your current setup
./snapshot restore           # preview what would be restored (safe, dry-run)
./snapshot restore --apply   # actually reinstall selected items
./snapshot --help            # show help
```

Backups are stored in a `snapshot-backup/` folder.

### Restoring

Run `restore` and you'll get an interactive checklist for apt packages and
snaps. Use arrow keys to move, **space** to tick, **enter** to confirm. Only the
items you select are installed.

## Safety

`restore` is **dry-run by default** — it shows what it *would* do and changes
nothing. It only makes real changes when you pass `--apply`. It uses `sudo` when
needed (and runs directly when already root).

## License

MIT — see [LICENSE](LICENSE).