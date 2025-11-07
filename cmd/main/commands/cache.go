package commands

import (
	"archivist/internal/app"
	"archivist/internal/cache"
	"archivist/internal/ui"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// NewCacheCommand creates the cache command with subcommands
func NewCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage analysis cache",
		Long:  "Manage the Redis cache for paper analysis results",
	}

	cmd.AddCommand(
		newCacheClearCommand(),
		newCacheStatsCommand(),
		newCacheListCommand(),
	)

	return cmd
}

func newCacheClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear [file1.pdf] [file2.pdf] ...",
		Short: "Clear cached analysis results",
		Long:  "Remove all cached analysis results from Redis, or specific papers if file paths are provided",
		Run:   runCacheClear,
	}
}

func newCacheStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show cache statistics",
		Long:  "Display statistics about cached analysis results",
		Run:   runCacheStats,
	}
}

func newCacheListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all cached papers",
		Long:  "Display all papers currently cached in Redis",
		Run:   runCacheList,
	}
}

func runCacheClear(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if !config.Cache.Enabled {
		ui.PrintWarning("Cache is not enabled in config")
		ui.PrintInfo("To enable cache, set cache.enabled: true in config/config.yaml")
		return
	}

	if config.Cache.Type != "redis" {
		ui.PrintError("Only Redis cache is supported for clearing")
		return
	}

	ui.PrintStage("Connecting to Redis", fmt.Sprintf("Connecting to %s", config.Cache.Redis.Addr))

	ctx := context.Background()
	ttl := time.Duration(config.Cache.TTL) * time.Hour
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		ttl,
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to Redis: %v", err))
		fmt.Println()
		ui.PrintInfo("Make sure Redis is running:")
		ui.ColorSubtle.Println("  sudo systemctl start redis")
		ui.ColorSubtle.Println("  # or")
		ui.ColorSubtle.Println("  redis-server")
		os.Exit(1)
	}
	defer redisCache.Close()

	ui.PrintSuccess("Connected to Redis")
	fmt.Println()

	// If specific files are provided, clear only those
	if len(args) > 0 {
		ui.PrintStage("Clearing Specific Papers", fmt.Sprintf("Removing cache for %d file(s)", len(args)))

		successCount := 0
		failCount := 0

		for _, filePath := range args {
			hash, err := fileutil.ComputeFileHash(filePath)
			if err != nil {
				ui.PrintError(fmt.Sprintf("❌ Failed to hash %s: %v", filePath, err))
				failCount++
				continue
			}

			err = redisCache.Delete(ctx, hash)
			if err != nil {
				ui.PrintError(fmt.Sprintf("❌ Failed to clear cache for %s: %v", filePath, err))
				failCount++
			} else {
				ui.PrintSuccess(fmt.Sprintf("✅ Cleared cache for: %s", filepath.Base(filePath)))
				successCount++
			}
		}

		fmt.Println()
		if successCount > 0 {
			ui.PrintSuccess(fmt.Sprintf("Successfully cleared %d cached entries", successCount))
		}
		if failCount > 0 {
			ui.PrintWarning(fmt.Sprintf("%d entries failed to clear", failCount))
		}
		fmt.Println()
		ui.ColorInfo.Println("💡 These papers will be analyzed fresh on next processing")
		fmt.Println()
		return
	}

	// Clear all cache entries
	count, err := redisCache.GetStats(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get stats: %v", err))
		os.Exit(1)
	}

	if count == 0 {
		ui.PrintInfo("Cache is already empty")
		return
	}

	ui.PrintWarning(fmt.Sprintf("Found %d cached entries", count))
	fmt.Println()
	ui.ColorWarning.Println("⚠️  This will permanently delete ALL cached analysis results!")
	fmt.Print("\nAre you sure? (yes/no): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		ui.PrintInfo("Cache clear cancelled")
		return
	}

	fmt.Println()
	ui.PrintStage("Clearing Cache", "Removing all cached entries")

	deleted, err := redisCache.Clear(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to clear cache: %v", err))
		os.Exit(1)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully cleared %d cached entries", deleted))
	fmt.Println()
	ui.ColorInfo.Println("💡 Next time you process papers, they will be analyzed fresh")
	fmt.Println()
}

func runCacheStats(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if !config.Cache.Enabled {
		ui.PrintWarning("Cache is not enabled in config")
		ui.PrintInfo("To enable cache, set cache.enabled: true in config/config.yaml")
		return
	}

	if config.Cache.Type != "redis" {
		ui.PrintError("Only Redis cache is supported for stats")
		return
	}

	ui.PrintStage("Connecting to Redis", fmt.Sprintf("Connecting to %s", config.Cache.Redis.Addr))

	ctx := context.Background()
	ttl := time.Duration(config.Cache.TTL) * time.Hour
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		ttl,
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to Redis: %v", err))
		fmt.Println()
		ui.PrintInfo("Make sure Redis is running:")
		ui.ColorSubtle.Println("  sudo systemctl start redis")
		ui.ColorSubtle.Println("  # or")
		ui.ColorSubtle.Println("  redis-server")
		os.Exit(1)
	}
	defer redisCache.Close()

	ui.PrintSuccess("Connected to Redis")
	fmt.Println()

	count, err := redisCache.GetStats(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get stats: %v", err))
		os.Exit(1)
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Println("                      CACHE STATISTICS                         ")
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	ui.ColorInfo.Printf("  📊 Total Cached Papers:  %d\n", count)
	ui.ColorInfo.Printf("  🕒 Cache TTL:            %d hours (%.0f days)\n",
		config.Cache.TTL, float64(config.Cache.TTL)/24)
	ui.ColorInfo.Printf("  🔗 Redis Address:        %s\n", config.Cache.Redis.Addr)
	ui.ColorInfo.Printf("  🗄️  Redis Database:       %d\n", config.Cache.Redis.DB)
	fmt.Println()

	if count == 0 {
		ui.ColorSubtle.Println("  💡 Cache is empty. Process some papers to populate the cache!")
	} else {
		ui.ColorSuccess.Println("  💡 Cache is active and saving API costs!")
		ui.ColorSubtle.Printf("     Estimated savings: %d Gemini API calls avoided\n", count)
	}
	fmt.Println()

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	ui.PrintInfo("To clear the cache, run: rph cache clear")
	fmt.Println()
}

func runCacheList(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if !config.Cache.Enabled {
		ui.PrintWarning("Cache is not enabled in config")
		ui.PrintInfo("To enable cache, set cache.enabled: true in config/config.yaml")
		return
	}

	if config.Cache.Type != "redis" {
		ui.PrintError("Only Redis cache is supported for listing")
		return
	}

	ui.PrintStage("Connecting to Redis", fmt.Sprintf("Connecting to %s", config.Cache.Redis.Addr))

	ctx := context.Background()
	ttl := time.Duration(config.Cache.TTL) * time.Hour
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		ttl,
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to Redis: %v", err))
		fmt.Println()
		ui.PrintInfo("Make sure Redis is running:")
		ui.ColorSubtle.Println("  sudo systemctl start redis")
		ui.ColorSubtle.Println("  # or")
		ui.ColorSubtle.Println("  redis-server")
		os.Exit(1)
	}
	defer redisCache.Close()

	ui.PrintSuccess("Connected to Redis")
	fmt.Println()

	ui.PrintStage("Fetching Cache", "Retrieving all cached papers")

	entries, err := redisCache.ListAll(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list cache: %v", err))
		os.Exit(1)
	}

	if len(entries) == 0 {
		ui.PrintInfo("Cache is empty")
		fmt.Println()
		ui.ColorSubtle.Println("  💡 Process some papers to populate the cache!")
		fmt.Println()
		return
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Printf("              CACHED PAPERS (%d)                           \n", len(entries))
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, entry := range entries {
		ui.ColorTitle.Printf("%d. %s\n", i+1, entry.PaperTitle)
		ui.ColorSubtle.Printf("   Hash:      %s...%s\n", entry.ContentHash[:8], entry.ContentHash[len(entry.ContentHash)-8:])
		ui.ColorSubtle.Printf("   Model:     %s\n", entry.ModelUsed)
		ui.ColorSubtle.Printf("   Cached:    %s (%s ago)\n",
			entry.CachedAt.Format("2006-01-02 15:04:05"),
			time.Since(entry.CachedAt).Round(time.Minute))
		ui.ColorInfo.Printf("   Size:      %d chars\n", len(entry.LatexContent))
		fmt.Println()
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	ui.PrintInfo("To clear specific papers, run: rph cache clear <file1.pdf> <file2.pdf>")
	ui.PrintInfo("To clear all cache, run: rph cache clear")
	fmt.Println()
}
