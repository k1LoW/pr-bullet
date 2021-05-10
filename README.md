# pr-bullet

`pr-bullet` is a tool for copying pull request to multiple repositories.

## Usage

First, create original pull request ( ex. `https://github.com/k1LoW/my-bullet/pull/3` ).

Then, specify the URL of the pull request and the repository where it is to be copied.

``` console
$ pr-bullet https://github.com/k1LoW/my-bullet/pull/3 k1LoW/tbls k1LoW/ndiag
Original pull request:
  Title ... Add allow-auto-merge label action using ghdag
  URL   ... https://github.com/k1LoW/my-bullet/pull/3
  Files ... 34
Target repositories:
  k1LoW/tbls, k1LoW/ndiag
Do you want to create pull requests? (y/n) [y]: y

Copying k1LoW/my-bullet pull request #3 to k1LoW/tbls ... https://github.com/k1LoW/tbls/pull/999 as draft
Copying k1LoW/my-bullet pull request #3 to k1LoW/ndiag ... https://github.com/k1LoW/ndiag/pull/333 as draft
$
```

### Required Environment variables

| Environment variable | Description | Default |
| --- | --- | --- |
| `GITHUB_TOKEN` | Personal access token | - |
| `GITHUB_API_URL` | API URL | `https://api.github.com` |

## Install

**deb:**

Use [dpkg-i-from-url](https://github.com/k1LoW/dpkg-i-from-url)

``` console
$ export PR-BULLET_VERSION=X.X.X
$ curl -L https://git.io/dpkg-i-from-url | bash -s -- https://github.com/k1LoW/pr-bullet/releases/download/v$PR-BULLET_VERSION/pr-bullet_$PR-BULLET_VERSION-1_amd64.deb
```

**RPM:**

``` console
$ export PR-BULLET_VERSION=X.X.X
$ yum install https://github.com/k1LoW/pr-bullet/releases/download/v$PR-BULLET_VERSION/pr-bullet_$PR-BULLET_VERSION-1_amd64.rpm
```

**apk:**

Use [apk-add-from-url](https://github.com/k1LoW/apk-add-from-url)

``` console
$ export PR-BULLET_VERSION=X.X.X
$ curl -L https://git.io/apk-add-from-url | sh -s -- https://github.com/k1LoW/pr-bullet/releases/download/v$PR-BULLET_VERSION/pr-bullet_$PR-BULLET_VERSION-1_amd64.apk
```

**homebrew tap:**

```console
$ brew install k1LoW/tap/pr-bullet
```

**manually:**

Download binary from [releases page](https://github.com/k1LoW/pr-bullet/releases)

**go get:**

```console
$ go get github.com/k1LoW/pr-bullet
```

**docker:**

```console
$ docker pull ghcr.io/k1low/pr-bullet:latest
```
