# GitFlow Mark 2 (Commands)

## Installation

 * Clone this repository.
 * `pip install -r requirements.txt --allow-unverified RBTools --allow-external RBTools`
 * Run `wget --no-check-certificate -q -O - https://github.com/realyze/gitflow2/raw/master/install.sh | sudo bash`

Then

 * Set `PIVOTAL_TOKEN` env var to your PT API token.
 * Create a `.workflowrc` file in your project's directory root and add: `PROJECT_ID='<your PR project id>'.`


## Commands
 * git flow2 feature start -> start a story
 * git flow2 post reviews -> rebase and post reviews for commits on a story branch
 * git flow2 update review -> Takes the last commit on current branch and attempts to find and update the corresponding review (looks for `story-id` line in the commit message).
