# Changelog

## [0.3.0](https://github.com/googleapis/google-cloud-go/compare/auth/v0.2.2...auth/v0.3.0) (2024-04-23)


### Features

* **auth/httptransport:** Add ability to customize transport ([#10023](https://github.com/googleapis/google-cloud-go/issues/10023)) ([72c7f6b](https://github.com/googleapis/google-cloud-go/commit/72c7f6bbec3136cc7a62788fc7186bc33ef6c3b3)), refs [#9812](https://github.com/googleapis/google-cloud-go/issues/9812) [#9814](https://github.com/googleapis/google-cloud-go/issues/9814)


### Bug Fixes

* **auth/credentials:** Error on bad file name if explicitly set ([#10018](https://github.com/googleapis/google-cloud-go/issues/10018)) ([55beaa9](https://github.com/googleapis/google-cloud-go/commit/55beaa993aaf052d8be39766afc6777c3c2a0bdd)), refs [#9809](https://github.com/googleapis/google-cloud-go/issues/9809)

## [0.2.2](https://github.com/googleapis/google-cloud-go/compare/auth/v0.2.1...auth/v0.2.2) (2024-04-19)


### Bug Fixes

* **auth:** Add internal opt to skip validation on transports ([#9999](https://github.com/googleapis/google-cloud-go/issues/9999)) ([9e20ef8](https://github.com/googleapis/google-cloud-go/commit/9e20ef89f6287d6bd03b8697d5898dc43b4a77cf)), refs [#9823](https://github.com/googleapis/google-cloud-go/issues/9823)
* **auth:** Set secure flag for gRPC conn pools ([#10002](https://github.com/googleapis/google-cloud-go/issues/10002)) ([14e3956](https://github.com/googleapis/google-cloud-go/commit/14e3956dfd736399731b5ee8d9b178ae085cf7ba)), refs [#9833](https://github.com/googleapis/google-cloud-go/issues/9833)

## [0.2.1](https://github.com/googleapis/google-cloud-go/compare/auth/v0.2.0...auth/v0.2.1) (2024-04-18)


### Bug Fixes

* **auth:** Default gRPC token type to Bearer if not set ([#9800](https://github.com/googleapis/google-cloud-go/issues/9800)) ([5284066](https://github.com/googleapis/google-cloud-go/commit/5284066670b6fe65d79089cfe0199c9660f87fc7))

## [0.2.0](https://github.com/googleapis/google-cloud-go/compare/auth/v0.1.1...auth/v0.2.0) (2024-04-15)

### Breaking Changes

In the below mentioned commits there were a few large breaking changes since the
last release of the module.

1. The `Credentials` type has been moved to the root of the module as it is
   becoming the core abstraction for the whole module.
2. Because of the above mentioned change many functions that previously
   returned a `TokenProvider` now return `Credentials`. Similarly, these
   functions have been renamed to be more specific.
3. Most places that used to take an optional `TokenProvider` now accept
   `Credentials`. You can make a `Credentials` from a `TokenProvider` using the
   constructor found in the `auth` package.
4. The `detect` package has been renamed to `credentials`. With this change some
   function signatures were also updated for better readability.
5. Derivative auth flows like `impersonate` and `downscope` have been moved to
   be under the new `credentials` package.

Although these changes are disruptive we think that they are for the best of the
long-term health of the module. We do not expect any more large breaking changes
like these in future revisions, even before 1.0.0. This version will be the
first version of the auth library that our client libraries start to use and
depend on.

### Features

* **auth/credentials/externalaccount:** Add default TokenURL ([#9700](https://github.com/googleapis/google-cloud-go/issues/9700)) ([81830e6](https://github.com/googleapis/google-cloud-go/commit/81830e6848ceefd055aa4d08f933d1154455a0f6))
* **auth:** Add downscope.Options.UniverseDomain ([#9634](https://github.com/googleapis/google-cloud-go/issues/9634)) ([52cf7d7](https://github.com/googleapis/google-cloud-go/commit/52cf7d780853594291c4e34302d618299d1f5a1d))
* **auth:** Add universe domain to grpctransport and httptransport ([#9663](https://github.com/googleapis/google-cloud-go/issues/9663)) ([67d353b](https://github.com/googleapis/google-cloud-go/commit/67d353beefe3b607c08c891876fbd95ab89e5fe3)), refs [#9670](https://github.com/googleapis/google-cloud-go/issues/9670)
* **auth:** Add UniverseDomain to DetectOptions ([#9536](https://github.com/googleapis/google-cloud-go/issues/9536)) ([3618d3f](https://github.com/googleapis/google-cloud-go/commit/3618d3f7061615c0e189f376c75abc201203b501))
* **auth:** Make package externalaccount public ([#9633](https://github.com/googleapis/google-cloud-go/issues/9633)) ([a0978d8](https://github.com/googleapis/google-cloud-go/commit/a0978d8e96968399940ebd7d092539772bf9caac))
* **auth:** Move credentials to base auth package ([#9590](https://github.com/googleapis/google-cloud-go/issues/9590)) ([1a04baf](https://github.com/googleapis/google-cloud-go/commit/1a04bafa83c27342b9308d785645e1e5423ea10d))
* **auth:** Refactor public sigs to use Credentials ([#9603](https://github.com/googleapis/google-cloud-go/issues/9603)) ([69cb240](https://github.com/googleapis/google-cloud-go/commit/69cb240c530b1f7173a9af2555c19e9a1beb56c5))


### Bug Fixes

* **auth/oauth2adapt:** Update protobuf dep to v1.33.0 ([30b038d](https://github.com/googleapis/google-cloud-go/commit/30b038d8cac0b8cd5dd4761c87f3f298760dd33a))
* **auth:** Fix uint32 conversion ([9221c7f](https://github.com/googleapis/google-cloud-go/commit/9221c7fa12cef9d5fb7ddc92f41f1d6204971c7b))
* **auth:** Port sts expires fix ([#9618](https://github.com/googleapis/google-cloud-go/issues/9618)) ([7bec97b](https://github.com/googleapis/google-cloud-go/commit/7bec97b2f51ed3ac4f9b88bf100d301da3f5d1bd))
* **auth:** Read universe_domain from all credentials files ([#9632](https://github.com/googleapis/google-cloud-go/issues/9632)) ([16efbb5](https://github.com/googleapis/google-cloud-go/commit/16efbb52e39ea4a319e5ee1e95c0e0305b6d9824))
* **auth:** Remove content-type header from idms get requests ([#9508](https://github.com/googleapis/google-cloud-go/issues/9508)) ([8589f41](https://github.com/googleapis/google-cloud-go/commit/8589f41599d265d7c3d46a3d86c9fab2329cbdd9))
* **auth:** Update protobuf dep to v1.33.0 ([30b038d](https://github.com/googleapis/google-cloud-go/commit/30b038d8cac0b8cd5dd4761c87f3f298760dd33a))

## [0.1.1](https://github.com/googleapis/google-cloud-go/compare/auth/v0.1.0...auth/v0.1.1) (2024-03-10)


### Bug Fixes

* **auth/impersonate:** Properly send default detect params ([#9529](https://github.com/googleapis/google-cloud-go/issues/9529)) ([5b6b8be](https://github.com/googleapis/google-cloud-go/commit/5b6b8bef577f82707e51f5cc5d258d5bdf90218f)), refs [#9136](https://github.com/googleapis/google-cloud-go/issues/9136)
* **auth:** Update grpc-go to v1.56.3 ([343cea8](https://github.com/googleapis/google-cloud-go/commit/343cea8c43b1e31ae21ad50ad31d3b0b60143f8c))
* **auth:** Update grpc-go to v1.59.0 ([81a97b0](https://github.com/googleapis/google-cloud-go/commit/81a97b06cb28b25432e4ece595c55a9857e960b7))

## 0.1.0 (2023-10-18)


### Features

* **auth:** Add base auth package ([#8465](https://github.com/googleapis/google-cloud-go/issues/8465)) ([6a45f26](https://github.com/googleapis/google-cloud-go/commit/6a45f26b809b64edae21f312c18d4205f96b180e))
* **auth:** Add cert support to httptransport ([#8569](https://github.com/googleapis/google-cloud-go/issues/8569)) ([37e3435](https://github.com/googleapis/google-cloud-go/commit/37e3435f8e98595eafab481bdfcb31a4c56fa993))
* **auth:** Add Credentials.UniverseDomain() ([#8654](https://github.com/googleapis/google-cloud-go/issues/8654)) ([af0aa1e](https://github.com/googleapis/google-cloud-go/commit/af0aa1ed8015bc8fe0dd87a7549ae029107cbdb8))
* **auth:** Add detect package ([#8491](https://github.com/googleapis/google-cloud-go/issues/8491)) ([d977419](https://github.com/googleapis/google-cloud-go/commit/d977419a3269f6acc193df77a2136a6eb4b4add7))
* **auth:** Add downscope package ([#8532](https://github.com/googleapis/google-cloud-go/issues/8532)) ([dda9bff](https://github.com/googleapis/google-cloud-go/commit/dda9bff8ec70e6d104901b4105d13dcaa4e2404c))
* **auth:** Add grpctransport package ([#8625](https://github.com/googleapis/google-cloud-go/issues/8625)) ([69a8347](https://github.com/googleapis/google-cloud-go/commit/69a83470bdcc7ed10c6c36d1abc3b7cfdb8a0ee5))
* **auth:** Add httptransport package ([#8567](https://github.com/googleapis/google-cloud-go/issues/8567)) ([6898597](https://github.com/googleapis/google-cloud-go/commit/6898597d2ea95d630fcd00fd15c58c75ea843bff))
* **auth:** Add idtoken package ([#8580](https://github.com/googleapis/google-cloud-go/issues/8580)) ([a79e693](https://github.com/googleapis/google-cloud-go/commit/a79e693e97e4e3e1c6742099af3dbc58866d88fe))
* **auth:** Add impersonate package ([#8578](https://github.com/googleapis/google-cloud-go/issues/8578)) ([e29ba0c](https://github.com/googleapis/google-cloud-go/commit/e29ba0cb7bd3888ab9e808087027dc5a32474c04))
* **auth:** Add support for external accounts in detect ([#8508](https://github.com/googleapis/google-cloud-go/issues/8508)) ([62210d5](https://github.com/googleapis/google-cloud-go/commit/62210d5d3e56e8e9f35db8e6ac0defec19582507))
* **auth:** Port external account changes ([#8697](https://github.com/googleapis/google-cloud-go/issues/8697)) ([5823db5](https://github.com/googleapis/google-cloud-go/commit/5823db5d633069999b58b9131a7f9cd77e82c899))


### Bug Fixes

* **auth/oauth2adapt:** Update golang.org/x/net to v0.17.0 ([174da47](https://github.com/googleapis/google-cloud-go/commit/174da47254fefb12921bbfc65b7829a453af6f5d))
* **auth:** Update golang.org/x/net to v0.17.0 ([174da47](https://github.com/googleapis/google-cloud-go/commit/174da47254fefb12921bbfc65b7829a453af6f5d))
