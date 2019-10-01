# Create a pull request

We're excited that you're considering contributing your change to the Grafana project! This document guides you through the process of creating a [pull request](https://help.github.com/en/articles/about-pull-requests/).

Before you create a pull request, make sure you've read [Contributing to Grafana](https://grafana.com/docs/contribute/overview/).

### Guidelines

- If think your pull request needs to be reviewed by a specific person, you can tag them in the description or in a comment. Tag a user by typing the `@` symbol followed by their GitHub username.

- Rebase to the master branch before submitting your pull request. If your pull request have conflicts, we'll ask you rebase your branch onto the master branch.

## Style guide

A well-written pull request increases the chance of getting your change accepted in a timely manner.

This style guide describes how to write good commit messages and descriptions for your pull requests.

### Commit message format

The Grafana guidelines for commit messages are based on [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/), with the following additions:

- Subject line must begin with the _area_ of the commit.
- Body is followed by an optional [keyword and issue reference](https://help.github.com/en/articles/closing-issues-using-keywords).

#### Area

Should be one of the following:

- **Build**: Changes to the build system, or external dependencies.
- **Chore**: Changes that don't affect functionality.
- **Dashboard**: Changes to the Dashboard feature.
- **Docs**: Changes that only affect documentation.
- **Explore**: Changes to the Explore feature.
- **Plugins**: Changes to any of the plugins.

Changes to data sources:

- AzureMonitor
- Graphite
- Prometheus

Changes to panels:

- GraphPanel
- SinglestatPanel
- TablePanel

**Example**

- `Build: Support publishing MSI to grafana.com`
- `Explore: Add Live option for supported data sources`
- `GraphPanel: Fix legend sorting issues`

### Pull request titles

Grafana _squashes_ all commits into one when a pull request gets accepted. The title of the pull request becomes the subject line of the squashed commit message. We still encourage contributors to write informative commit messages, as they will be included in the Git commit body.

The pull request title is used when we generate change logs for releases.

The title for the pull request should be descriptive, as we use the title when we generate the change logs for releases.

- The title for the pull request uses the same format as the subject line in the commit message.

## Code review

Once you've created a pull request, the next step is to have someone review your change. A review is a learning opportunity for both the reviewer and the author of the pull request.

Read [How to do a code review](https://google.github.io/eng-practices/review/reviewer/) to learn more about some best practices for code reviews.
