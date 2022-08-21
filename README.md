# Semserv - A minimal semgrep rule server

Semserv is a web server used to serve semgrep rules from any GitHub repositories,
it can effectively work as a mirror of the official semgrep rule registry or as
a registry that serves custom rules.

## How to use?

For now, repositories to pull from must be defined inside `main.go` in the
`rulesets` variable.

Once the desired semgrep rule repositories are set up, then simply run the
following command to start the server:

```bash
$ go run .
```

Semgrep can then be setup to fetch from semserv by setting the `SEMGREP_URL`
environment variable.

Assuming that the URL on which semserv is running is `https://foo.com`, the
following semgrep call should work:

```bash
$ SEMGREP_URL="https://foo.com/" semgrep --config "p/rhps"
```

## Limitations

At this point in time Semserv is very barebones, thus it comes with quite a bit
of limitations:

* `--config auto` doesn't work.
* Rules are fetched from GitHub on-the-fly using the GitHub API which can be good
  or bad depending on how you look at it.
* New repositories to expose must be defined in the source code.
* Not all ruleset paths are supported
* Other stuff that I can't remember
