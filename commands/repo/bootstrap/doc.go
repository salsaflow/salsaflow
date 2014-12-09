/*
Generate initial local SalsaFlow configuration.

  salsaflow repo bootstrap -issue_tracker=ISSUE_TRACKER
                           -code_review_tool=CODE_REVIEW_TOOL
                           [-skeleton=SKELETON]

Description

This command can be used to generate initial local SalsaFlow configuration
so that the repository can be set up quickly.

The -issue_tracker and -code_review_tool flags must be supplied to tell
SalsaFlow what modules you are going to use for the project. It affects
the local configuration file template that is generated.

The -skeleton flag can be used to quickly set up the remaining configuration
for SalsaFlow, most importantly the custom scripts. When supplied with
OWNER/REPO, SalsaFlow will get the GitHub repository specified by that value
and it will copy the content into the local metadata directory. This can be
easily used to share custom scripts by making a skeleton for every distinct
project type.

Steps

This command goes through the following steps:

  1. Take the flags and write a configuration file template into
    .salsaflow/config.yml. This template needs to be filled in, but the keys
    to be filled are clearly visible.
  2. We are done in case -skeleton is not specified.
  3. Clone the given skeleton repository into the local cache, which is located
     in the current user's home directory. The repository is just pulled in case
     it already exists. Once the content is available, it is copied into
     the local metadata directory, .salsaflow.
*/
package bootstrapCmd
