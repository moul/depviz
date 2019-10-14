module moul.io/depviz

go 1.13

require (
	github.com/brianloveswords/airtable v0.0.0-20180329193050-a39294038dd9
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/docgen v1.0.5
	github.com/go-chi/render v1.0.1
	github.com/google/go-github/v28 v28.1.1
	github.com/jinzhu/gorm v1.9.11
	github.com/lib/pq v1.2.0
	github.com/mattn/go-sqlite3 v1.11.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/xanzy/go-gitlab v0.20.1
	go.uber.org/zap v1.10.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	gopkg.in/yaml.v2 v2.2.4
	moul.io/graphman v1.5.0
	moul.io/graphman/viz v0.0.0-20190925205035-97b8bdad4639
	moul.io/multipmuri v1.8.0
	moul.io/zapgorm v0.0.0-20190706070406-8138918b527b
)

replace github.com/brianloveswords/airtable => github.com/moul/brianloveswords-airtable v0.0.0-20191014120838-8b07ee6d33b2
