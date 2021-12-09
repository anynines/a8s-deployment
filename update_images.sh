#!/bin/bash

# TODO: Switch to "sed" when copying inside github action

set -o errexit
set -o nounset
set -o pipefail

VERSIONED_IMGS="postgresql-operator:v0.9.0 backup-manager:v0.7.0 service-binding-controller:v0.5.0"
for VERSIONED_IMG in $VERSIONED_IMGS
do
    # Extract image name and version as separate variables
    IMG=$(echo $VERSIONED_IMG | cut -d ':' -f 1)
    NEW_VERSION=$(echo $VERSIONED_IMG | cut -d ':' -f 2)

    GET_IMG_FIELD_REGEXP="s/^[[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$IMG:\(v[\.[:digit:]]\{1,\}\)\"\{0,1\}$/\1/p"
    UPDATE_IMG_FIELD_REGEXP="s/^\([[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$IMG:\)v[\.[:digit:]]\{1,\}\(\"\{0,1\}\)$/\1$NEW_VERSION\2/"

    # This regex could be stricter. For example, it doesn't check that in the semver version there are no leading zeros
    # if the semver number has more than one digit (e.g. v01.1.1 would pass). But right now it's enough and it's much
    # more readable this way than in "stricter" formats.
    CURRENT_VERSION=$(gsed -n $GET_IMG_FIELD_REGEXP "deploy/a8s/$IMG.yaml")
    # This check might misbehave if one of the semver versions has a version number with leading zeros and more than one
    # digit. But that should never happen since we control those version numbers and there's no reason why we should 
    # end up having leading zeros in version numbers with more than one digit.
    if [[ "$NEW_VERSION" > "$CURRENT_VERSION" ]]
    then
        gsed -i $UPDATE_IMG_FIELD_REGEXP "deploy/a8s/$IMG.yaml"
        # TODO: Uncomment before pushing real version.
        # git add "deploy/a8s/$IMG.yaml"
        # git commit -m "Bump $IMG to $NEW_VERSION"
    else
        echo "$IMG current version is $CURRENT_VERSION, most recent version found is $NEW_VERSION, no update needed"
    fi
done

# TODO: Handle logging
# TODO: Handle opensearch-dashboards
# TODO: Refactor to reuse the function
