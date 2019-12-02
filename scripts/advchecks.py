#!/usr/bin/env python

import json

def find_lgtms(comments):
    commenters = []
    for comment in comments:
        body = comment["body"].strip()
        if body == "/lgtm":
            commenters.append("%s(%s)" % (comment["user"]["login"], comment["user"]["id"]))
    return commenters

def find_reviewers(pulls):
    reviewers = []
    for reviewer in pulls["requested_reviewers"]:
        reviewers.append("%s(%s)" % (reviewer["login"], reviewer["id"]))
    return reviewers
 
if __name__ == '__main__':
    import sys

    if len(sys.argv) < 3:
        print(sys.argv[0], "<pull>", "<comment>")
        sys.exit(-1)

    with open(sys.argv[1]) as pullfile:
        pulls = json.load(pullfile)
        with open(sys.argv[2]) as commentfile:
            comments = json.load(commentfile)
            rvs = find_reviewers(pulls)
            cms = find_lgtms(comments)
            if len(rvs) == 0:
                print("No reviwer is assigned, give up check...")
                sys.exit(-1)
            print("Assigned reviwers: %s" % ", ".join(rvs))
            print("Lgtm reviwers: %s" % ", ".join(cms))
            req = []
            for rv in rvs:
                if rv not in cms:
                    req.append(rv)
            if len(req) > 0:
                print("Reviewers %s needs /lgtm" % ",".join(req))
                sys.exit(-1)
