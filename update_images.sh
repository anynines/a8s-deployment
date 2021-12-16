#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

new_version_is_newer () {
    # Replace "." and "-" with " " from the versions so that it becomes easier to compare each
    # version token from the new version to the corresponding one from the current version.
    local NEW_VERSION=$(sed "s/[\.v-]/ /g" <<< $1)
    local CURRENT_VERSION=$(sed "s/[\.v-]/ /g" <<< $2)

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
    local COMPONENT=$1
    local NEW_VERSION=$2
    local MANIFEST=$3

    # Prepare sed expression to extract the current version of the component from its yaml manifest.
    # The regexp isn't strict: it matches the image version, but it'll match also incorrect
    # formats. I started with an extremely precise regexp but it was overly long and complex, so I
    # opted for allowing some incorrect formats for simplicity's sake. Since we control the parsed
    # manifests we can have strong guarantees that the versions will be in the right formats, so
    # there should be no issues.
    local GET_VERSION_SED_CMD="s/^[[:space:]-]\{1,\}image:[[:space:]].\{1,\}\/$COMPONENT:\(v[\.[:digit:]-]\{1,\}\)\"\{0,1\}$/\1/p"
    local CURRENT_VERSION=$(sed -n $GET_VERSION_SED_CMD $MANIFEST)

    if new_version_is_newer "$NEW_VERSION" "$CURRENT_VERSION"
    then
        # Prepare sed expression to update the version of the image in its yaml manifest. The regexp
        # isn't strict: it matches the image version, but it'll match also incorrect formats. I
        # started with an extremely precise regexp but it was overly long and complex, so I opted
        # for allowing some incorrect formats for simplicity's sake. Since we control the parsed
        # manifests we can have strong guarantees that the versions will be in the right formats, so
        # there should be no issues.
        # TODO: Switch to "sed" when copying inside github action
        local UPDATE_VERSION_SED_CMD="s/^\([[:space:]-]\{1,\}image:[[:space:]].\{1,\}\/$COMPONENT:\)v[\.[:digit:]-]\{1,\}\(\"\{0,1\}\)$/\1$NEW_VERSION\2/"
        sed -i $UPDATE_VERSION_SED_CMD $MANIFEST
        # TODO: Uncomment before pushing real version.
        #echo "Bump $COMPONENT to $NEW_VERSION"
        git add "$MANIFEST"
        git commit -m "Bump $COMPONENT to $NEW_VERSION"
    else
        echo "Current version of $COMPONENT is $CURRENT_VERSION, most recent version found is $NEW_VERSION, no update needed"
    fi
}

main () {
    local VERSIONED_IMGS="postgresql-operator:v0.8.1 backup-manager:v1.2.0 service-binding-controller:v0.3.0 fluentd:v1.12.0-1.0-3.3.0 opensearch-dashboards:v1.0.1-1.0.0"
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

main
