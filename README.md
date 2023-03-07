# Redmine bot for Mattermost

Redmine issue name expander for Mattermost.

Bot parses the issue numbers (`#54321`), the links to issue (`http://redmine.host.com/issues/54321`) and displays the issue name with attributes:
* project
* tracker
* author
* assigned to
* category
* version

## Run

### From sources

Create .env file from .env.dist, then:
```
make build run
```

The binary will be installed at `./bin/mattermost-redmine-bot`.

### As a container

Use available [Docker image](https://hub.docker.com/r/akuzia/mattermost-redmine-bot)