#!/bin/bash

# TODO: Switch to "sed" when copying inside github action

set -o errexit
set -o nounset
set -o pipefail

update_fluentd_img_and_commit () {
    local NEW_VERSION=$1
    local MANIFEST="deploy/logging/collection-infrastructure/fluentd-aggregator.yaml"

    local GET_VERSION_SED_CMD="s/^[[:space:]-]\{1,\}image:[[:space:]].\{1,\}\/fluentd:\(v[\.[:digit:]-]\{1,\}\)\"\{0,1\}$/\1/p"
    local CURRENT_VERSION=$(gsed -n $GET_VERSION_SED_CMD $MANIFEST)

    if [[ "$NEW_VERSION" > "$CURRENT_VERSION" ]]
    then
        local UPDATE_VERSION_SED_CMD="s/^\([[:space:]-]\{1,\}image:[[:space:]].\{1,\}\/fluentd:\)v[\.[:digit:]-]\{1,\}\(\"\{0,1\}\)$/\1$NEW_VERSION\2/"
        gsed -i "$UPDATE_VERSION_SED_CMD" "$MANIFEST"
        # TODO: Uncomment before pushing real version.
        echo "Bump fluentd-aggregator to $NEW_VERSION"
        # git add "$MANIFEST"
        # git commit -m "Bump fluentd-aggregator to $NEW_VERSION"
    else
        echo "fluentd-aggregator current version is $CURRENT_VERSION, most recent version found is $NEW_VERSION, no update needed"
    fi
}

# Right now the core components are the PostgreSQL Operator, the Backup Manager and the Service
# Binding Controller.
update_core_component_img_and_commit () {
    local COMPONENT=$1
    local NEW_VERSION=$2
    local MANIFEST="deploy/a8s/$COMPONENT.yaml"

    local GET_VERSION_SED_CMD="s/^[[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$COMPONENT:\(v[\.[:digit:]]\{1,\}\)\"\{0,1\}$/\1/p"
    local CURRENT_VERSION=$(gsed -n $GET_VERSION_SED_CMD $MANIFEST)

    if [[ "$NEW_VERSION" > "$CURRENT_VERSION" ]]
    then
        local UPDATE_VERSION_SED_CMD="s/^\([[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$COMPONENT:\)v[\.[:digit:]]\{1,\}\(\"\{0,1\}\)$/\1$NEW_VERSION\2/"
        gsed -i "$UPDATE_VERSION_SED_CMD" "$MANIFEST"
        # TODO: Uncomment before pushing real version.
        echo "Bump $COMPONENT to $NEW_VERSION"
        # git add "$MANIFEST"
        # git commit -m "Bump $COMPONENT to $NEW_VERSION"
    else
        echo "$COMPONENT current version is $CURRENT_VERSION, most recent version found is $NEW_VERSION, no update needed"
    fi
}

main () {
    local VERSIONED_IMGS="postgresql-operator:v0.9.0 backup-manager:v0.7.0 service-binding-controller:v0.5.0 fluentd:v0.12.3-1.0-1.1.0"
    for VERSIONED_IMG in $VERSIONED_IMGS
    do
        # Extract image name and version as separate variables
        local IMG=$(echo $VERSIONED_IMG | cut -d ':' -f 1)
        local NEW_VERSION=$(echo $VERSIONED_IMG | cut -d ':' -f 2)

        # If needed, update the image version in the yaml manifests and commit each update
        # individually to easily pinpoint which update broke things in case tests fail.
        if [[ "$IMG" == "fluentd" ]]
        then
            update_fluentd_img_and_commit $NEW_VERSION
        elif [[ "$IMG" == "opensearch-dashboards" ]]
        then
            echo "opensearch-dashboards"
        else
            update_core_component_img_and_commit $IMG $NEW_VERSION
        fi
    done
}

main

# TODO: Handle logging
# TODO: Handle opensearch-dashboards
