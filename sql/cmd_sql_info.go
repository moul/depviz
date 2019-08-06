package sql

/*
func (cmd *dbCommand) dbInfoCommand() *cobra.Command {
	cc := &cobra.Command{
		Use: "info",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			return dbInfo(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	return cc
}


func dbInfo(opts *dbOptions) error {
	fmt.Printf("database: %q\n", dbPath)
	for _, model := range model.AllModels {
		var count int
		tableName := db.NewScope(model).TableName()
		if err := db.Model(model).Count(&count).Error; err != nil {
			log.Printf("failed to get count for %q: %v", tableName, err)
			continue
		}
		fmt.Printf("stats: %-20s %3d\n", tableName, count)
	}
	return nil
}
*/
