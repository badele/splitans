# Changelog

## [0.10.1](https://github.com/badele/splitans/compare/v0.10.0...v0.10.1) (2026-03-09)


### Bug Fixes

* N metadata parsing ([#45](https://github.com/badele/splitans/issues/45)) ([5a5e864](https://github.com/badele/splitans/commit/5a5e864e11a606372dcaa545de216f394e8fccd9))
* **neotex:** computing !N with -K flag ([#44](https://github.com/badele/splitans/issues/44)) ([92c197c](https://github.com/badele/splitans/commit/92c197c816897c228460988611335b41ac7a6077))


### Miscellaneous

* remove release multiple package support ([#42](https://github.com/badele/splitans/issues/42)) ([089c10e](https://github.com/badele/splitans/commit/089c10ec051bb29b80b5bf95d7d7403291cfe758))

## [0.10.0](https://github.com/badele/splitans/compare/v0.9.0...v0.10.0) (2026-03-05)


### Features

* add -K keep-trailing-lines option ([#41](https://github.com/badele/splitans/issues/41)) ([3b131ed](https://github.com/badele/splitans/commit/3b131ed3855744589281fea943ec2cefaa44fe02))
* update VGA output option command ([#39](https://github.com/badele/splitans/issues/39)) ([53a04b5](https://github.com/badele/splitans/commit/53a04b510ffa7e58c79de1715a268af071edbdf4))

## [0.9.0](https://github.com/badele/splitans/compare/v0.8.0...v0.9.0) (2026-02-28)


### Features

* add ANSI 1990 legacy mode and neotex rebuild hooks ([#37](https://github.com/badele/splitans/issues/37)) ([ed7c3f3](https://github.com/badele/splitans/commit/ed7c3f3701ba52e0a4f893155ef65e2a8a20526d))
* add content bounds ([#9](https://github.com/badele/splitans/issues/9)) ([df94c68](https://github.com/badele/splitans/commit/df94c685cb7fc9a379715e5e1caef11e61909443))
* add hyperlink and foreground and background hover color ([#18](https://github.com/badele/splitans/issues/18)) ([2d3df06](https://github.com/badele/splitans/commit/2d3df062c754f22307d718adcea46c9ebe40a354))
* add neotex logo ([#35](https://github.com/badele/splitans/issues/35)) ([ee0eb51](https://github.com/badele/splitans/commit/ee0eb5182fad2d6a2b8ddad5b87cb6b3a49ea701))
* add palette support ([#32](https://github.com/badele/splitans/issues/32)) ([6ad51f9](https://github.com/badele/splitans/commit/6ad51f915c4a908b36c8e4a2e0399c5ad4f7b885))
* add public splitans API and improve ANSI handling ([#2](https://github.com/badele/splitans/issues/2)) ([f42c07c](https://github.com/badele/splitans/commit/f42c07c56e4ec5d0f61ef2e2d7c9df3b13ba7216))
* add SAUCE metadata support for import and export ([#14](https://github.com/badele/splitans/issues/14)) ([7ac6f38](https://github.com/badele/splitans/commit/7ac6f38a8556f7f0572ef40c9987f1fd384d15f8))
* define default SAUCE date ([#29](https://github.com/badele/splitans/issues/29)) ([b1b74b9](https://github.com/badele/splitans/commit/b1b74b987a0a124beeec1a75f2131dc33aa1bdd4))
* **exporter:** add inline output mode for single-line export ([#5](https://github.com/badele/splitans/issues/5)) ([19e9af5](https://github.com/badele/splitans/commit/19e9af5c0e4b28392bd3ff70c6df255897240a77))
* first release ([#1](https://github.com/badele/splitans/issues/1)) ([c0db96d](https://github.com/badele/splitans/commit/c0db96d6a393575d1419146032cf99bc2725f1c3))
* fix neotex parser ([#25](https://github.com/badele/splitans/issues/25)) ([32cef95](https://github.com/badele/splitans/commit/32cef95ed9ca62349f7cc3337d51d90e2b5f156b))
* **hyperlink:** implement OSC 8 hyperlink handling with CP437 legacy mode ([#16](https://github.com/badele/splitans/issues/16)) ([1d867e1](https://github.com/badele/splitans/commit/1d867e152dcedbc4d01163e76a6dca27669e828f))
* ignore null & space character ([#22](https://github.com/badele/splitans/issues/22)) ([f610e5c](https://github.com/badele/splitans/commit/f610e5c0e10f9b88386f93001aa5a0218354525b))
* **neotex:** parse !TW width header and propagate to output width ([#3](https://github.com/badele/splitans/issues/3)) ([6d74f8e](https://github.com/badele/splitans/commit/6d74f8e4014e87ce613f99c5395568efa87d20cb))
* update metadata ([#21](https://github.com/badele/splitans/issues/21)) ([efac7c6](https://github.com/badele/splitans/commit/efac7c6a42d1985fffeef09e90a6226006e3f177))
* update neotex versioning ([#20](https://github.com/badele/splitans/issues/20)) ([71e8555](https://github.com/badele/splitans/commit/71e8555334f2beb556cdece07762fe9bc143d00c))
* update rising error ([#30](https://github.com/badele/splitans/issues/30)) ([9d81a11](https://github.com/badele/splitans/commit/9d81a116683315e79e502fba3dc046145b669aa8))
* **virtualterminal:** add GetContentBounds ([#8](https://github.com/badele/splitans/issues/8)) ([ee420e4](https://github.com/badele/splitans/commit/ee420e49967c642a42335b7f10beb51271d5aece))


### Bug Fixes

* **cli:** quote default values in struct tags ([#6](https://github.com/badele/splitans/issues/6)) ([b174f3f](https://github.com/badele/splitans/commit/b174f3fa60ad72218ddcc2487bc11e8f3270497c))
* **neotex:** calculate true width correctly in inline mode ([#7](https://github.com/badele/splitans/issues/7)) ([318fbfd](https://github.com/badele/splitans/commit/318fbfd61bfcb11f42190d424b71c6d7c3f5cf68))
* release please ([#34](https://github.com/badele/splitans/issues/34)) ([edbf246](https://github.com/badele/splitans/commit/edbf246b4ac797e81ceffc960636860b3e7104aa))
* release please components ([#31](https://github.com/badele/splitans/issues/31)) ([0d3f3f9](https://github.com/badele/splitans/commit/0d3f3f9183bfef34120f76cad5bfa72c64ea7522))
* sauce dimension ([#26](https://github.com/badele/splitans/issues/26)) ([f2fc396](https://github.com/badele/splitans/commit/f2fc39659a3241c34eb2dbe838600c14f6fade70))
* **virtualterminal:** ignore CR/LF after soft wrap ([#4](https://github.com/badele/splitans/issues/4)) ([14e2817](https://github.com/badele/splitans/commit/14e2817dfa6071731c0dd70eea67d686e74a1f6b))


### Documentation

* Add development guide ([#11](https://github.com/badele/splitans/issues/11)) ([5aa0a8b](https://github.com/badele/splitans/commit/5aa0a8b209f0efa5272bcf5285a56be298b57f68))


### Miscellaneous

* fix release please version ([#27](https://github.com/badele/splitans/issues/27)) ([c8e622d](https://github.com/badele/splitans/commit/c8e622d4d7d44fcae47d5e639b767218ff3c0a05))
* **main:** release 0.2.0 ([#13](https://github.com/badele/splitans/issues/13)) ([2f9dc5f](https://github.com/badele/splitans/commit/2f9dc5f3b312c008a5529ac29fb275fc44bcde35))
* **main:** release 0.3.0 ([#15](https://github.com/badele/splitans/issues/15)) ([b714529](https://github.com/badele/splitans/commit/b71452986192f062045e09d4b25e2cb8cc98519c))
* **main:** release 0.4.0 ([#17](https://github.com/badele/splitans/issues/17)) ([d60dd68](https://github.com/badele/splitans/commit/d60dd68e25abecf6c0661cf1c5f9daf1225e15e1))
* **main:** release 0.5.0 ([#19](https://github.com/badele/splitans/issues/19)) ([4c71c8b](https://github.com/badele/splitans/commit/4c71c8b985b1f6fbc40bc2ef0efa819bf31c59b8))
* **main:** release 0.6.0 ([#28](https://github.com/badele/splitans/issues/28)) ([e89e925](https://github.com/badele/splitans/commit/e89e925cc3a13fbd8fa85ef21576904c5d32eda2))
* release main ([#33](https://github.com/badele/splitans/issues/33)) ([14d1f16](https://github.com/badele/splitans/commit/14d1f16ee3111906bf7306bb95050470cf9e212b))
* release main ([#36](https://github.com/badele/splitans/issues/36)) ([55fdb14](https://github.com/badele/splitans/commit/55fdb1419bc661490fd67f7903244418563468fa))

## [0.8.0](https://github.com/badele/splitans/compare/v0.7.0...v0.8.0) (2026-02-26)


### Features

* add neotex logo ([#35](https://github.com/badele/splitans/issues/35)) ([ee0eb51](https://github.com/badele/splitans/commit/ee0eb5182fad2d6a2b8ddad5b87cb6b3a49ea701))

## [0.7.0](https://github.com/badele/splitans/compare/v0.6.0...v0.7.0) (2026-02-26)


### Features

* add palette support ([#32](https://github.com/badele/splitans/issues/32)) ([6ad51f9](https://github.com/badele/splitans/commit/6ad51f915c4a908b36c8e4a2e0399c5ad4f7b885))
* define default SAUCE date ([#29](https://github.com/badele/splitans/issues/29)) ([b1b74b9](https://github.com/badele/splitans/commit/b1b74b987a0a124beeec1a75f2131dc33aa1bdd4))
* update rising error ([#30](https://github.com/badele/splitans/issues/30)) ([9d81a11](https://github.com/badele/splitans/commit/9d81a116683315e79e502fba3dc046145b669aa8))


### Bug Fixes

* release please ([#34](https://github.com/badele/splitans/issues/34)) ([edbf246](https://github.com/badele/splitans/commit/edbf246b4ac797e81ceffc960636860b3e7104aa))
* release please components ([#31](https://github.com/badele/splitans/issues/31)) ([0d3f3f9](https://github.com/badele/splitans/commit/0d3f3f9183bfef34120f76cad5bfa72c64ea7522))

## [0.6.0](https://github.com/badele/splitans/compare/v0.5.0...v0.6.0) (2026-02-14)


### Features

* fix neotex parser ([#25](https://github.com/badele/splitans/issues/25)) ([32cef95](https://github.com/badele/splitans/commit/32cef95ed9ca62349f7cc3337d51d90e2b5f156b))
* ignore null & space character ([#22](https://github.com/badele/splitans/issues/22)) ([f610e5c](https://github.com/badele/splitans/commit/f610e5c0e10f9b88386f93001aa5a0218354525b))
* update metadata ([#21](https://github.com/badele/splitans/issues/21)) ([efac7c6](https://github.com/badele/splitans/commit/efac7c6a42d1985fffeef09e90a6226006e3f177))
* update neotex versioning ([#20](https://github.com/badele/splitans/issues/20)) ([71e8555](https://github.com/badele/splitans/commit/71e8555334f2beb556cdece07762fe9bc143d00c))


### Bug Fixes

* sauce dimension ([#26](https://github.com/badele/splitans/issues/26)) ([f2fc396](https://github.com/badele/splitans/commit/f2fc39659a3241c34eb2dbe838600c14f6fade70))


### Miscellaneous

* fix release please version ([#27](https://github.com/badele/splitans/issues/27)) ([c8e622d](https://github.com/badele/splitans/commit/c8e622d4d7d44fcae47d5e639b767218ff3c0a05))

## [0.5.0](https://github.com/badele/splitans/compare/v0.4.0...v0.5.0) (2026-02-03)


### Features

* add hyperlink and foreground and background hover color ([#18](https://github.com/badele/splitans/issues/18)) ([2d3df06](https://github.com/badele/splitans/commit/2d3df062c754f22307d718adcea46c9ebe40a354))

## [0.4.0](https://github.com/badele/splitans/compare/v0.3.0...v0.4.0) (2026-01-24)


### Features

* **hyperlink:** implement OSC 8 hyperlink handling with CP437 legacy mode ([#16](https://github.com/badele/splitans/issues/16)) ([1d867e1](https://github.com/badele/splitans/commit/1d867e152dcedbc4d01163e76a6dca27669e828f))

## [0.3.0](https://github.com/badele/splitans/compare/v0.2.0...v0.3.0) (2026-01-19)


### Features

* add SAUCE metadata support for import and export ([#14](https://github.com/badele/splitans/issues/14)) ([7ac6f38](https://github.com/badele/splitans/commit/7ac6f38a8556f7f0572ef40c9987f1fd384d15f8))

## [0.2.0](https://github.com/badele/splitans/compare/v0.1.0...v0.2.0) (2026-01-18)


### Features

* add content bounds ([#9](https://github.com/badele/splitans/issues/9)) ([df94c68](https://github.com/badele/splitans/commit/df94c685cb7fc9a379715e5e1caef11e61909443))
* add public splitans API and improve ANSI handling ([#2](https://github.com/badele/splitans/issues/2)) ([f42c07c](https://github.com/badele/splitans/commit/f42c07c56e4ec5d0f61ef2e2d7c9df3b13ba7216))
* **exporter:** add inline output mode for single-line export ([#5](https://github.com/badele/splitans/issues/5)) ([19e9af5](https://github.com/badele/splitans/commit/19e9af5c0e4b28392bd3ff70c6df255897240a77))
* first release ([#1](https://github.com/badele/splitans/issues/1)) ([c0db96d](https://github.com/badele/splitans/commit/c0db96d6a393575d1419146032cf99bc2725f1c3))
* **neotex:** parse !TW width header and propagate to output width ([#3](https://github.com/badele/splitans/issues/3)) ([6d74f8e](https://github.com/badele/splitans/commit/6d74f8e4014e87ce613f99c5395568efa87d20cb))
* **virtualterminal:** add GetContentBounds ([#8](https://github.com/badele/splitans/issues/8)) ([ee420e4](https://github.com/badele/splitans/commit/ee420e49967c642a42335b7f10beb51271d5aece))


### Bug Fixes

* **cli:** quote default values in struct tags ([#6](https://github.com/badele/splitans/issues/6)) ([b174f3f](https://github.com/badele/splitans/commit/b174f3fa60ad72218ddcc2487bc11e8f3270497c))
* **neotex:** calculate true width correctly in inline mode ([#7](https://github.com/badele/splitans/issues/7)) ([318fbfd](https://github.com/badele/splitans/commit/318fbfd61bfcb11f42190d424b71c6d7c3f5cf68))
* **virtualterminal:** ignore CR/LF after soft wrap ([#4](https://github.com/badele/splitans/issues/4)) ([14e2817](https://github.com/badele/splitans/commit/14e2817dfa6071731c0dd70eea67d686e74a1f6b))


### Documentation

* Add development guide ([#11](https://github.com/badele/splitans/issues/11)) ([5aa0a8b](https://github.com/badele/splitans/commit/5aa0a8b209f0efa5272bcf5285a56be298b57f68))
