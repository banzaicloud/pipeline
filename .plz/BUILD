genrule(
    name = "go.mod",
    outs = ["go.mod"],
    cmd = [
        """export GO=$(cat $(pwd | sed "s/plz-out\/.*//")/go.mod | grep '^go .*$') && cat > $OUT << EOF
module plz-out

$GO
EOF
      """
    ],
    labels = [
        "link:plz-out/"
    ],
    visibility = ["PUBLIC"],
)
