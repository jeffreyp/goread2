# Project Overview

This project is a multi-user web-based RSS/ATOM feed reader that is
deployed on Google App Engine and Google Data Store.

# Overall Rules

* Begin each request by first planning its implementation, showing a
  todo list, and confirming the approach.
* For each change, make relevant updates to both documentation and tests.

# Environment Guidelines

* Prefer `python3` to `python`.
* "Production" refers to a deployment to Google App Engine/Google Data
  Store. "Local" refers to a local deployment using sqlite3.
** ALL code changes must support and be tested to confirm operable on both.

# Debugging Guidelines

* Debugging statements should be used as a last resort to minimize
  unnecessary production deployments.

# Buildig and Testing

* Everything you need should be in the Makefile.

# Version Control

* ALWAYS build and test before committing. Builds must succeed and ALL
  tests must pass before committing.
** This includes running the linter.

