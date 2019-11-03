# Clipman

**WARNING: this is the development branch for v2. It's not yet production ready.**

A basic clipboard manager for Wayland, with support for persisting copy buffers after an application exits.

## Installing

Requirements:

- a windows manager that uses `wlr-data-control`, like Sway and other wlroots-based WMs.
- wofi

Run `cargo install` inside this folder after installing the devel tools for Rust.

<!-- Archlinux users can find a PKGBUILD [here](https://aur.archlinux.org/packages/clipman/). -->

## Usage

Run the binary in your Sway session by adding `exec clipman watch` (or `exec clipman watch 1>> PATH/TO/LOGFILE 2>&1 &` to log errors) at the beginning of your config.

To query the history and select items, run the binary as `clipman pick`. You can assign it to a keybinding: `bindsym $mod+h exec clipman pick`.

To remove items from history, `clipman clear` and `clipman clear --all`.

For more options: `clipman -h`.

## Versions

This projects follows SemVer conventions.
