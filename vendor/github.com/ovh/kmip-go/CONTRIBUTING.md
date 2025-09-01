# Contributing to kmip-go
 
This project accepts contributions. In order to contribute, you should
pay attention to a few things:
 
1. your code must be unit-tested
2. your code must be documented
3. your work must be signed (see below)
4. you may contribute through GitHub Pull Requests
5. Commits follow [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/) style (see below)
 
# Submitting Modifications
 
The contributions should be submitted through Github Pull Requests
and follow the DCO which is defined below.
 
# Licensing for new files
 
kmip-go is licensed under a Apache 2.0 license. Anything
contributed to kmip-go must be released under this license.
 
When introducing a new file into the project, please make sure it has a
copyright header making clear under which license it's being released.

# Commit Message Guidelines

We have very precise rules over how our git commit messages can be formatted.  This leads to **more
readable messages** that are easy to follow when looking through the **project history**.  But also,
we use the git commit messages to **generate the change log**.

Commits descriptions MUST follow [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/) style.

## Commit Message Format
Each commit message consists of a **header**, a **body** and a **footer**.  The header has a special
format that includes a **type**, a **scope** and a **subject**:

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

The **header** is mandatory and the **scope** of the header is optional.

Any line of the commit message cannot be longer 100 characters! This allows the message to be easier
to read on BitBucket as well as in various git tools.

Samples:

```
docs(changelog): update changelog to v12.26.5
```
```
fix(release): need to depend on latest foobar

It fixes the critical CVE #1233
```

## Revert
If the commit reverts a previous commit, it should begin with `revert: `, followed by the header of the reverted commit. In the body it should say: `This reverts commit <hash>.`, where the hash is the SHA of the commit being reverted.

## Type
Must be one of the following:

* **build**: Changes that affect the build system or external dependencies
* **ci**: Changes to our CI configuration files and scripts
* **docs**: Documentation only changes
* **feat**: A new feature
* **fix**: A bug fix
* **perf**: A code change that improves performance
* **refactor**: A code change that neither fixes a bug nor adds a feature
* **style**: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
* **test**: Adding missing tests or correcting existing tests

## Scope
The scope should be the name of the package affected (as perceived by the person reading the changelog generated from commit messages.)
There are currently a few exceptions to the "use package name" rule:

* **changelog**: used for updating the release notes in CHANGELOG.md
* none/empty string: useful for `style`, `test` and `refactor` changes that are done across all packages (e.g. `style: add missing semicolons`)

## Subject
The subject contains a succinct description of the change:

* use the imperative, present tense: "change" not "changed" nor "changes"
* don't capitalize the first letter
* no dot (.) at the end

## Body
Just as in the **subject**, use the imperative, present tense: "change" not "changed" nor "changes".
The body should include the motivation for the change and contrast this with previous behavior.

## Footer
The footer should contain any information about **Breaking Changes**

**Breaking Changes** should start with the word `BREAKING CHANGE:` with a space or two newlines. The rest of the commit message is then used for this.

 
# Developer Certificate of Origin (DCO)
 
To improve tracking of contributions to this project we will use a
process modeled on the modified DCO 1.1 and use a "sign-off" procedure
on patches that are being emailed around or contributed in any other
way.
 
The sign-off is a simple line at the end of the explanation for the
patch, which certifies that you wrote it or otherwise have the right
to pass it on as an open-source patch.  The rules are pretty simple:
if you can certify the below:
 
By making a contribution to this project, I certify that:
 
(a) The contribution was created in whole or in part by me and I have
    the right to submit it under the open source license indicated in
    the file; or
 
(b) The contribution is based upon previous work that, to the best of
    my knowledge, is covered under an appropriate open source License
    and I have the right under that license to submit that work with
    modifications, whether created in whole or in part by me, under
    the same open source license (unless I am permitted to submit
    under a different license), as indicated in the file; or
 
(c) The contribution was provided directly to me by some other person
    who certified (a), (b) or (c) and I have not modified it.
 
(d) The contribution is made free of any other party's intellectual
    property claims or rights.
 
(e) I understand and agree that this project and the contribution are
    public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
 
 
then you just add a line saying
 
    Signed-off-by: Random J Developer <random@example.org>
 
using your real name (sorry, no pseudonyms or anonymous contributions.)