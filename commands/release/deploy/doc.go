/*
Deploy a release into production.

  salsaflow release deploy

Description

Deploy the current staging environment into production.

Steps

This command goes through the following steps:

  1. The issue tracker is checked to make sure the release
     is accepted and that it can be actually released.
  2. The stable branch is reset to point to the staging branch.
  3. Version is bumped for the stable branch.
  4. The stable branch is tagged with a release tag.
  5. The staging branch is reset to the current release branch
     in case there is already another release started.
  6. Everything is pushed to the remote repository.
*/
package deployCmd
