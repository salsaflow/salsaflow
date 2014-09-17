set -e

. git-flow2-rainbow.sh
. git-flow2-spinner.sh
. git-flow2-common.sh


PIVOTAL_TOKEN=$(ensure_pt "token" "global") || {
  echo "${PIVOTAL_TOKEN}"
  exit 1
}
export PIVOTAL_TOKEN="${PIVOTAL_TOKEN}"


PT_ROOT="https://www.pivotaltracker.com/services/v5"
PT_AUTH_HDR="X-TrackerToken: ${PIVOTAL_TOKEN}"


function pt_set_me_as_story_owner {
  local project_id=$1
  local story_id=$2

  pt_user_id=$(\
    curl -s -X GET -H "${PT_AUTH_HDR}" "${PT_ROOT}/me" 2>/dev/null | \
    underscore extract "id" --outfmt text 2>/dev/null) || true

  if [[ -z "${pt_user_id}" ]]; then
    echo "${__bold}ERROR: Could not determine your Pivotal Tracker user id.${__normal}"
    return 1
  fi

  # Set the story owner field to me.
  # PT reports errors in JSON output with HTTP 200 so we have to parse what we get.
  __out=$(\
    curl -X PUT -H "${PT_AUTH_HDR}" \
      -H "Content-Type: application/json" \
      -d "{\"owner_ids\": [${pt_user_id}]}" \
      --silent \
      "${PT_ROOT}/projects/${project_id}/stories/${story_id}") || {
    echo "Something went wront when trying to access Pivotal Tracker server."
    return 1
  }

  # Check if there were errors.
  pt_error=$(\
    echo ${__out} | underscore select ".error" --outfmt text)

  if [[ -n "${pt_error}" ]]; then
    echo ${pt_error}
    return 1
  fi

  return 0
}


function pt_start_story {
  local project_id=$1
  local story_id=$2

  # Instruct PT backend to update the story.
  echo
  __out=$(pivotal_tools start story ${story_id} --project-id ${project_id}) || {
    echo "${out}"
    return 1
  }

  return 0
}

