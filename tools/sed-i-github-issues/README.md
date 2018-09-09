# set-i-github-issues

Replace multiple issue bodies at once, with a regexp.

## Example

    GITHUB_TOKEN=xxx sed-i-github-issues -owner=moul -repo=roadmap -pattern "child of" -replace "blocks"
