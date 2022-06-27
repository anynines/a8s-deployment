#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# This script is invoked by a github action that runs within an ubuntu VM. But it might be invoked
# manually by developers on their macos work laptop for testing. This is problematic because the
# script uses "sed", and sed's syntax is different between linux and macos. So we assume that the
# developer testing from macos also has "gsed" (the macos version of linux's sed; i.e. gsed on
# macos has the same syntax as sed on linux) installed on his machine. The following if selects
# gsed as the command to use if this script runs on macos, while picks "sed" otherwise (it assumes
# that if the OS isn't macos then it's linux).
SED="sed"
if [[ $OSTYPE == 'darwin'* ]]
then
    SED="gsed"
fi
readonly SED

# new_version_is_newer assumes that the two versions that it receives as arguments are in the
# same format, and will fail (in some cases, silently) if that's not the case.
new_version_is_newer () {
    # Replace ".", "-" and "v" with " " in the versions so that it becomes easier to compare each
    # version token from the new version to the corresponding token from the current version. What
    # do I mean by token? For example I see a semver 2 version as:
    # v<major-token>.<minor-token>.<patch-token>.
    local NEW_VERSION=$($SED "s/[\.v-]/ /g" <<< $1)
    local CURRENT_VERSION=$($SED "s/[\.v-]/ /g" <<< $2)

    # From a string containing all the tokens of a version to an array where each item represents
    # a single token (in descending order of priority), to ease comparison between new and current
    # version.
    local NEW_VERSION_TOKENS=( $NEW_VERSION )
    local CURRENT_VERSION_TOKENS=( $CURRENT_VERSION )

    # Now, compare each token between new and current version in descending order of priority, to
    # establish which version is newer.
    for i in "${!NEW_VERSION_TOKENS[@]}"
    do
        if [ ${NEW_VERSION_TOKENS[$i]} -gt ${CURRENT_VERSION_TOKENS[$i]} ]
        then
            return 0
        fi
        if [ ${NEW_VERSION_TOKENS[$i]} -lt ${CURRENT_VERSION_TOKENS[$i]} ]
        then
            return 1
        fi
    done

    return 1
}

ensure_image_is_fresh_and_commit () {
    local IMG=$1
    local NEW_VERSION=$2
    local MANIFEST=$3

    # Prepare sed expression to extract the current version of the image from its yaml manifest.
    # The regexp isn't strict: it matches the image version, but it'll match also incorrect
    # formats. I started with an extremely precise regexp but it was overly long and complex, so I
    # opted for allowing some incorrect formats for simplicity's sake. Since we control the parsed
    # manifests we can have strong guarantees that the versions will be in the right formats, so
    # there should be no issues. Notice that the group that captures the version matches more than
    # just semver 2 versions, because we have some images (fluentd and opensearch-dashboards) that
    # don't follow semver 2.
    local GET_VERSION_SED_CMD="s/^[[:space:]-]\{1,\}image:[[:space:]].\{1,\}\/$IMG:\(v[\.[:digit:]-]\{1,\}\)\"\{0,1\}$/\1/p"
    local CURRENT_VERSION=$($SED -n $GET_VERSION_SED_CMD $MANIFEST)

    if new_version_is_newer "$NEW_VERSION" "$CURRENT_VERSION"
    then
        # Prepare sed expression to update the version of the image in its yaml manifest. The regexp
        # isn't strict: it matches the image version, but it'll match also incorrect formats. I
        # started with an extremely precise regexp but it was overly long and complex, so I opted
        # for allowing some incorrect formats for simplicity's sake. Since we control the parsed
        # manifests we can have strong guarantees that the versions will be in the right formats, so
        # there should be no issues. Notice that the group that captures the version matches more
        # than just semver 2 versions, because we have some images (fluentd and
        # opensearch-dashboards) that don't follow semver 2.
        local UPDATE_VERSION_SED_CMD="s/^\([[:space:]-]\{1,\}image:[[:space:]].\{1,\}\/$IMG:\)v[\.[:digit:]-]\{1,\}\(\"\{0,1\}\)$/\1$NEW_VERSION\2/"
        $SED -i $UPDATE_VERSION_SED_CMD $MANIFEST
        git add "$MANIFEST"
        git commit -m "Bump $IMG to $NEW_VERSION"
    else
        echo "Current version of $IMG is $CURRENT_VERSION, most recent version found is $NEW_VERSION, no update needed"
    fi
}

main () {
    local VERSIONED_IMGS="$1"
    for VERSIONED_IMG in $VERSIONED_IMGS
    do
        # Extract image name and version as separate variables
        local IMG=$(echo $VERSIONED_IMG | cut -d ':' -f 1)
        local NEW_VERSION=$(echo $VERSIONED_IMG | cut -d ':' -f 2)

        # Each image needs to be updated in a yaml manifest with an ad-hoc name (i.e. there's no
        # regular pattern), so we have to branch and manually build the manifest name differently
        # for each component.
        if [[ "$IMG" == "fluentd" ]]
        then
            local MANIFEST="deploy/logging/collection-infrastructure/fluentd-aggregator.yaml"
        elif [[ "$IMG" == "opensearch-dashboards" ]]
        then
            local MANIFEST="deploy/logging/dashboard/opensearch-dashboards.yaml"
        else
            local MANIFEST="deploy/a8s/$IMG.yaml"
        fi

        # If needed, update the image version in the relevant yaml manifests and commit each update
        # individually to easily pinpoint which update broke things in case tests fail.
        ensure_image_is_fresh_and_commit $IMG $NEW_VERSION $MANIFEST
    done
}

main "$1"
