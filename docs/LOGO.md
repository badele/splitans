## generate

```bash
nix shell github:badele/splitans nixpkgs#ansilove
curl "https://codef-ansi-logo-maker-api.santo.fr/api.php?text=splitans&font=1192&spacing=1&spacesize=6&vary=1" > logo.ans
splitans -W 89 logo.ans > logo.neo
splitans -f neotex -E cp437 -F ansi -S logo.neo > logo.ans
ansilove -c 89 -o logo.png logo.ans
```

## Source

- Tools
  - [ansilove](https://github.com/ansilove/ansilove)
  - [CODEF](https://n0namen0.github.io/CODEF_Ansi_Logo_Maker/)
