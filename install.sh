#!/bin/bash

# Does this need to be smarter for each host OS?
if [ -z "$INSTALL_PREFIX" ] ; then
  INSTALL_PREFIX="/usr/local/bin"
fi

if [ -z "$REPO_NAME" ] ; then
  REPO_NAME="SalsaFlow"
fi

if [ -z "$REPO_HOME" ] ; then
  REPO_HOME="http://github.com/salsita/${REPO_NAME}.git"
fi

BASH_DIR="gitflow2"
EXEC_FILES="git-flow2 git-flow2-feature-start git-flow2-review-update git-flow2-post-reviews git-flow2-init"
SCRIPT_FILES="git-flow2-spinner.sh git-flow2-rainbow.sh git-flow2-common.sh git-flow2-pivotal.sh"
GO_FILES="git-trunk git-trunk-hook-commit-msg"

GIT_TRUNK_VERSION="0.2.0"

echo "### SalsaFlow no-make installer ###"

case "$1" in
  uninstall)
    echo "Uninstalling SalsaFlow from $INSTALL_PREFIX"
    if [ -d "$INSTALL_PREFIX" ] ; then
      for script_file in $SCRIPT_FILES $EXEC_FILES $GO_FILES; do
        echo "rm -vf $INSTALL_PREFIX/$script_file"
        rm -vf "$INSTALL_PREFIX/$script_file"
      done
    else
      echo "The '$INSTALL_PREFIX' directory was not found."
      echo "Do you need to set INSTALL_PREFIX ?"
    fi
    exit
    ;;
  help)
    echo "Usage: [environment] installer.sh [install|uninstall]"
    echo "Environment:"
    echo "   INSTALL_PREFIX=$INSTALL_PREFIX"
    echo "   REPO_HOME=$REPO_HOME"
    echo "   REPO_NAME=$REPO_NAME"
    exit
    ;;
  *)
    echo "Installing SalsaFlow to $INSTALL_PREFIX"
    if [ -d "$REPO_NAME" -a -d "$REPO_NAME/.git" ] ; then
      echo "Using existing repo: $REPO_NAME"
    else
      echo "Cloning repo from GitHub to $REPO_NAME"
      git clone --branch master --single-branch "$REPO_HOME" "$REPO_NAME"
    fi
    echo "Installing python requirements"
    pip install -r "$REPO_NAME/$BASH_DIR/requirements.txt" \
      --allow-external RBTools --allow-unverified RBTools
    if [[ "$?" -ne "0" ]]; then
      exit 1
    fi
    install -v -d -m 0755 "$INSTALL_PREFIX"
    for exec_file in $EXEC_FILES ; do
      install -v -m 0755 "$REPO_NAME/$BASH_DIR/$exec_file" "$INSTALL_PREFIX"
    done
    for script_file in $SCRIPT_FILES ; do
      install -v -m 0644 "$REPO_NAME/$BASH_DIR/$script_file" "$INSTALL_PREFIX"
    done
    GIT_TRUNK="salsaflow.${GIT_TRUNK_VERSION}.darwin-amd64-osx10.9"
    if [[ ! -f "${GIT_TRUNK}.zip" ]]; then
      wget "https://github.com/salsita/${REPO_NAME}/releases/download/v${GIT_TRUNK_VERSION}/${GIT_TRUNK}.zip"
    fi
    unzip -o "${GIT_TRUNK}.zip"
    for go_binary in $GO_FILES ; do
      install -v -m 0755 "${GIT_TRUNK}/${go_binary}" "${INSTALL_PREFIX}"
    done
    exit
    ;;
esac
