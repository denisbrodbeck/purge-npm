# purge-npm finds all your `node_modules` folders and burns them with fire

<p align="center">
  <a href="https://godoc.org/github.com/denisbrodbeck/purge-npm"><img src="https://godoc.org/github.com/denisbrodbeck/purge-npm?status.svg" alt="GoDoc"></a>
  <a href="https://goreportcard.com/report/github.com/denisbrodbeck/purge-npm"><img src="https://goreportcard.com/badge/github.com/denisbrodbeck/purge-npm" alt="Go Report Card"></a>
</p>

Tested on: Linux, OS X, Windows

## Why

Well, why not? Who needs *28.000 files* in a pristine `create-react-app`, anyways?

I work on the web, I try new stuff, I abandon failed projects and I **always** forget to clean up my `node_modules` directories after switching projects. And I always forget the correct syntax for find/bash/powershell.

## What

* tiny command line app written in go
* expects a root directory - defaults to current directory
* searches below the root directory all directories named `node_modules` using a breadth first approach
* and **deletes those suckerz**

## How

```sh
go get github.com/denisbrodbeck/purge-npm

purge-npm ~/code/
  /home/luke/code/react-app/node_modules
  /home/luke/code/vuejs-app/node_modules
  /home/luke/code/gatsby-site/node_modules
  /home/luke/code/corp/web/blub/node_modules
  ...
  # ~1.500.000 deleted files later 7GB disc space is freed
```

All available flags:

```text
purge-npm [--dry] [<path>]
Flags:
  --dry  <bool>     output found directories only - do not remove
```

All exit codes:

```text
0: success (each removed folder is printed to stdout)
1: execution error (see stderr)
2: cli usage error (see stderr)
```

Possible failures:

* stack-overflow because of the recursive directory walking function - but then you've already got waaay bigger problems than this tool failing
* file permissions
* invalid cli usage (impossible path, etc.)

## License

The MIT License (MIT) — [Denis Brodbeck](https://github.com/denisbrodbeck). Please have a look at the [LICENSE](LICENSE) for more details.
