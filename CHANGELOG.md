# Changelog

## [1.2.0](https://github.com/sygmaprotocol/sygma-relayer/compare/v1.1.4...v1.2.0) (2023-02-24)


### Features

* substrate integration ([#114](https://github.com/sygmaprotocol/sygma-relayer/issues/114)) ([7f8f0b9](https://github.com/sygmaprotocol/sygma-relayer/commit/7f8f0b972b5e849bcf1b2197ee2becef1906b541))
* support erc20 handler response ([#109](https://github.com/sygmaprotocol/sygma-relayer/issues/109)) ([2d902ba](https://github.com/sygmaprotocol/sygma-relayer/commit/2d902baffa09812e757d461fdb3562b5b3f477b1))
* switch tss lib ([#115](https://github.com/sygmaprotocol/sygma-relayer/issues/115)) ([673792c](https://github.com/sygmaprotocol/sygma-relayer/commit/673792cf7137ecf61dad8f0d8ea059f835702c99))

## [1.1.4](https://github.com/sygmaprotocol/sygma-relayer/compare/v1.1.3...v1.1.4) (2023-02-17)


### Bug Fixes

* starting block with latest flag ([#104](https://github.com/sygmaprotocol/sygma-relayer/issues/104)) ([72907dc](https://github.com/sygmaprotocol/sygma-relayer/commit/72907dc1c2f2c2c922ac7042c57a1bf26b3e0ccd))

## [1.1.3](https://github.com/sygmaprotocol/sygma-relayer/compare/v1.1.2...v1.1.3) (2023-02-02)


### Miscellaneous

* bump core version ([#101](https://github.com/sygmaprotocol/sygma-relayer/issues/101)) ([70a6d85](https://github.com/sygmaprotocol/sygma-relayer/commit/70a6d85bf25c99da9fd7bd100aa49aaeb17695d6))
* refactor event handler logging ([#99](https://github.com/sygmaprotocol/sygma-relayer/issues/99)) ([d47ccc1](https://github.com/sygmaprotocol/sygma-relayer/commit/d47ccc1403d1090cf0ffd5af3932c40e6b92d3b1))

## [1.1.2](https://github.com/sygmaprotocol/sygma-relayer/compare/v1.1.1...v1.1.2) (2023-02-01)


### Miscellaneous

* refactor communication health check ([#97](https://github.com/sygmaprotocol/sygma-relayer/issues/97)) ([a8039a2](https://github.com/sygmaprotocol/sygma-relayer/commit/a8039a201f78bf9648c442e5bc573398e241699f))

## [1.1.1](https://github.com/sygmaprotocol/sygma-relayer/compare/v1.1.0...v1.1.1) (2023-01-30)


### Bug Fixes

* fix topology resolution ([#92](https://github.com/sygmaprotocol/sygma-relayer/issues/92)) ([fefefbe](https://github.com/sygmaprotocol/sygma-relayer/commit/fefefbe3afaca68f07092af0fdee3f11aa9195ab))
* health endpoint not starting ([#95](https://github.com/sygmaprotocol/sygma-relayer/issues/95)) ([2957ab0](https://github.com/sygmaprotocol/sygma-relayer/commit/2957ab0c9f9f13261404e521f7deafa273bdff49))


### Miscellaneous

* add explicit error log to lvlDB connection ([#87](https://github.com/sygmaprotocol/sygma-relayer/issues/87)) ([80f0c65](https://github.com/sygmaprotocol/sygma-relayer/commit/80f0c658f7c648ac386c7eb7a001509af1183128))
* added jinja2 render ([#89](https://github.com/sygmaprotocol/sygma-relayer/issues/89)) ([2731c1c](https://github.com/sygmaprotocol/sygma-relayer/commit/2731c1c7defe6fb37627913595717e9188ff8bc1))
* execute health check when health endpoint called ([#94](https://github.com/sygmaprotocol/sygma-relayer/issues/94)) ([84cb3e5](https://github.com/sygmaprotocol/sygma-relayer/commit/84cb3e5dc2a6d6f047c1e55bec43fd4815eaa5b1))
* fix ref error ([#90](https://github.com/sygmaprotocol/sygma-relayer/issues/90)) ([dc79c6e](https://github.com/sygmaprotocol/sygma-relayer/commit/dc79c6ed8f32aa10dac02af9cf6e1329ec857612))

## [1.1.0](https://github.com/sygmaprotocol/sygma-relayer/compare/v1.0.0...v1.1.0) (2023-01-23)


### Features

* Add substrate chain type ([#42](https://github.com/sygmaprotocol/sygma-relayer/issues/42)) ([f2c40c0](https://github.com/sygmaprotocol/sygma-relayer/commit/f2c40c012a57cec744901263c1f95c14210026d4))
* Add substrate connection ([#51](https://github.com/sygmaprotocol/sygma-relayer/issues/51)) ([627e8fa](https://github.com/sygmaprotocol/sygma-relayer/commit/627e8fa863c986b7869160d04feb3f4f31317095))
* add substrate event listener module ([#46](https://github.com/sygmaprotocol/sygma-relayer/issues/46)) ([d46030b](https://github.com/sygmaprotocol/sygma-relayer/commit/d46030b180fb3fc6fe54470858d39d572b2a9e5d))
* Add substrate message-handler ([#65](https://github.com/sygmaprotocol/sygma-relayer/issues/65)) ([ec71361](https://github.com/sygmaprotocol/sygma-relayer/commit/ec71361da6e69bb53828e813c0ba124f5a9ef385))
* copy existing workflows to deploy to TESTNET ([#59](https://github.com/sygmaprotocol/sygma-relayer/issues/59)) ([63fea85](https://github.com/sygmaprotocol/sygma-relayer/commit/63fea85e423f4a05a5605be6479cf842377e9eae))
* fetch topology from ipfs ([#40](https://github.com/sygmaprotocol/sygma-relayer/issues/40)) ([8daa7e2](https://github.com/sygmaprotocol/sygma-relayer/commit/8daa7e2c78eb0de7b8f4826d10ca8a7db9e44c62))
* Generate libp2p key command ([#36](https://github.com/sygmaprotocol/sygma-relayer/issues/36)) ([27103c1](https://github.com/sygmaprotocol/sygma-relayer/commit/27103c19b6f90a9420f84336eb93a6279198f631))
* Implement deposit(fungibleTransfer) event handler ([#55](https://github.com/sygmaprotocol/sygma-relayer/issues/55)) ([23d2479](https://github.com/sygmaprotocol/sygma-relayer/commit/23d24799300e1e3de215c815a9b8be2798abb5a1))
* integrate datadog in testnet relayers ([#85](https://github.com/sygmaprotocol/sygma-relayer/issues/85)) ([206d325](https://github.com/sygmaprotocol/sygma-relayer/commit/206d325cd421d39618f435644a207290728a37d7))
* process permissionless generic  messages sequentially ([#41](https://github.com/sygmaprotocol/sygma-relayer/issues/41)) ([6933acb](https://github.com/sygmaprotocol/sygma-relayer/commit/6933acb4b5d907bafca9f744a8d8de4bf26964b8))
* release pipeline ([48a96eb](https://github.com/sygmaprotocol/sygma-relayer/commit/48a96eb3994717482396bf2eac7b2b687abdb0bb))
* shared configuration ([#84](https://github.com/sygmaprotocol/sygma-relayer/issues/84)) ([eace2ad](https://github.com/sygmaprotocol/sygma-relayer/commit/eace2ad5084c68e46bc9df81bd0ccfb57688d96d))
* support for permissionless generic format v2 ([#60](https://github.com/sygmaprotocol/sygma-relayer/issues/60)) ([3343708](https://github.com/sygmaprotocol/sygma-relayer/commit/3343708bdf8c443f024ea67fbbfce0a2f72b0a82))


### Bug Fixes

* improve tss process sessionID logging ([#71](https://github.com/sygmaprotocol/sygma-relayer/issues/71)) ([6ad9b21](https://github.com/sygmaprotocol/sygma-relayer/commit/6ad9b219c30058e4eae0dbfb39f5f40e037a5ced))
* libp2p streams not closed after usage ([#54](https://github.com/sygmaprotocol/sygma-relayer/issues/54)) ([60e65f8](https://github.com/sygmaprotocol/sygma-relayer/commit/60e65f8773a7d84fe633ee9586d2150eb438b21c))
* release not creating PR ([056be93](https://github.com/sygmaprotocol/sygma-relayer/commit/056be93531a79a678fa61e872dbcc4738b37b023))
* remove deprecated libp2p packages ([#50](https://github.com/sygmaprotocol/sygma-relayer/issues/50)) ([6320494](https://github.com/sygmaprotocol/sygma-relayer/commit/6320494479dbabfc934c55ba7db576dae73fa344))
* resources id issue ([#74](https://github.com/sygmaprotocol/sygma-relayer/issues/74)) ([168b80e](https://github.com/sygmaprotocol/sygma-relayer/commit/168b80ea8ab08e43871d4348ff102dddc0818a10))


### Miscellaneous

* 'changed actor to repository_owner' ([#79](https://github.com/sygmaprotocol/sygma-relayer/issues/79)) ([2607d74](https://github.com/sygmaprotocol/sygma-relayer/commit/2607d7462f467e8a5ae70e0ad17b91e5cb1b996e))
* 'fixed typo on matrix job' remove duplicates & exposed data ([#78](https://github.com/sygmaprotocol/sygma-relayer/issues/78)) ([7e01cac](https://github.com/sygmaprotocol/sygma-relayer/commit/7e01cac72ddb157ed72a2b0d8198e68b54ae8ab3))
* 'fixed typo' remove duplicates & exposed data ([#77](https://github.com/sygmaprotocol/sygma-relayer/issues/77)) ([e96f1c4](https://github.com/sygmaprotocol/sygma-relayer/commit/e96f1c47e71462e9e34a0246ba74c987b2f8cd9b))
* add code quality analysis by github ([#44](https://github.com/sygmaprotocol/sygma-relayer/issues/44)) ([b327429](https://github.com/sygmaprotocol/sygma-relayer/commit/b32742993b0bf7870c55bc7bcb94383070140a9f))
* add shibuya to relayers ([#67](https://github.com/sygmaprotocol/sygma-relayer/issues/67)) ([84b76e0](https://github.com/sygmaprotocol/sygma-relayer/commit/84b76e0942bfb2a5a07e6ea0f25c807adb32559a))
* bump solidity version ([#23](https://github.com/sygmaprotocol/sygma-relayer/issues/23)) ([db53bd4](https://github.com/sygmaprotocol/sygma-relayer/commit/db53bd4a56dc1ffb9658d31e834a752396fcadae))
* changed github_token ([#80](https://github.com/sygmaprotocol/sygma-relayer/issues/80)) ([d693b67](https://github.com/sygmaprotocol/sygma-relayer/commit/d693b6732eaf79e369c204b0211033311563bd6e))
* expand logging on relayer start ([#86](https://github.com/sygmaprotocol/sygma-relayer/issues/86)) ([5bb0cbc](https://github.com/sygmaprotocol/sygma-relayer/commit/5bb0cbca7fb3dbfcfa89a5d87bea8484ac58a90a))
* fixed forbidden error ([#83](https://github.com/sygmaprotocol/sygma-relayer/issues/83)) ([1dd5fdd](https://github.com/sygmaprotocol/sygma-relayer/commit/1dd5fdd537efef9f3fc4279443e8e1636eec822b))
* publish binaries on release ([#39](https://github.com/sygmaprotocol/sygma-relayer/issues/39)) ([adf8af2](https://github.com/sygmaprotocol/sygma-relayer/commit/adf8af2c44be6a129ca6b5472de0800c09b436a8))
* remove duplicates & exposed data ([#76](https://github.com/sygmaprotocol/sygma-relayer/issues/76)) ([a6db67b](https://github.com/sygmaprotocol/sygma-relayer/commit/a6db67badb05c7ad2d5ea02913fa32ecd2f90b77))
* remove unused functions ([#64](https://github.com/sygmaprotocol/sygma-relayer/issues/64)) ([96a9b96](https://github.com/sygmaprotocol/sygma-relayer/commit/96a9b963da39ce6409c3858451f7ad02159c7bc9))
