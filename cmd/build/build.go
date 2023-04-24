package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/pkg/buildx"
	"github.com/errordeveloper/imagine/pkg/config"
	"github.com/errordeveloper/imagine/pkg/git"
	"github.com/errordeveloper/imagine/pkg/rebuilder"
	"github.com/errordeveloper/imagine/pkg/recipe"
	"github.com/errordeveloper/imagine/pkg/registry"
)

type Flags struct {
	*config.CommonFlags
	Builder string

	Force bool

	SummaryFormat string
	SummaryOutput string
}

const (
	stateDir = ".imagine" // TODO(post-mvp): make this repo config field

	defaultPlatform       = "linux/amd64"
	defaultUpstreamBranch = "origin/master"
)

func BuildCmd() *cobra.Command {

	flags := &Flags{
		CommonFlags: &config.CommonFlags{},
	}

	cmd := &cobra.Command{
		Use:   "build",
		Short: "build and test image(s) using a config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitBuildCmd(cmd); err != nil {
				return err
			}
			return flags.RunBuildCmd()
		},
	}

	flags.Register(cmd)

	cmd.Flags().StringVar(&flags.Builder, "builder", "", "use a global buildx builder instead of creating one")

	cmd.Flags().BoolVar(&flags.Force, "force", false, "force rebuilding")

	cmd.Flags().StringVar(&flags.SummaryFormat, "summary-format", "auto", "format of build summary as in simple 'text' format, or CSV 'lines', as well as 'json' or 'yaml'")
	cmd.Flags().StringVar(&flags.SummaryOutput, "summary-output", "-", "write build summary to either '-' (stdout) or a given file path")

	return cmd
}

func (f *Flags) InitBuildCmd(cmd *cobra.Command) error {
	return f.Validate()
}

func (f *Flags) RunBuildCmd() error {
	initialWD, err := os.Getwd()
	if err != nil {
		return err
	}

	stateDirPath := filepath.Join(initialWD, stateDir)

	repo, err := git.New(initialWD)
	if err != nil {
		return err
	}

	bcPath := f.Config
	if filepath.IsAbs(bcPath) {
		bcPath, err = filepath.Rel(initialWD, bcPath)
		if err != nil {
			return err
		}
	}

	bc, bcData, err := config.Load(bcPath)
	if err != nil {
		return err
	}

	if f.Debug {
		fmt.Printf("loaded config: %#v\n", *bc)
		fmt.Printf(".Spec.WithBuildInstructions: %#v\n", bc.Spec.WithBuildInstructions)
		for i, variant := range bc.Spec.Variants {
			fmt.Printf(".Spec.Vairiants[%d]: {Name:%q, With:%#v}\n", i, variant.Name, variant.With)
		}
	}

	// TODO:
	// - [x] new tagging convetion
	// - [ ] implement metadata labels
	//    - [x] build config contents and tree hash
	//	  -	[x] context tree hash
	//    - [ ] (post-mvp) additonal info
	//			NB: any of this will not carry any guarantees, it should be trated as hints;
	//			since it is unlikely to make build results reproducible, it should be stricly optional
	//       - [ ] remote URL (with an option to specify prefer remote name(s))
	//       - [ ] optionally resolve commit hashes for non-tagged commits
	//		 - [ ] resolve commit hashes for tagged commits
	//		 - [ ] optionaly resolve branch names
	//		 - [ ] grab metadata from GitHub event, when used in GitHub Actions
	// - [x] store config as loaded from disk
	// - [x] rebuilder must check all variants
	// - [x] compose multi-target recipe directly from the config
	//    - [x] there should be just one invocation of bake
	// - [ ] (post-mvp) --repo-config for repo-wide config
	// - [?] write exact image names at the end of the build
	//    - [ ] (post-mvp) review if `bake --metadata-file` output is sufficient
	// 			or some if it might need a wrapper (e.g. to match API style)
	//    - [ ] (post-mvp) review if summary/metadata aids 'index.json'
	//    - [x] ... as plain text summary to stdout
	//    - [x] (post-mvp) ... as JSON/YAML
	//    - [x] (post-mvp) as plain text file with basic CSV formatting
	//    - [ ] (post-mvp) as plain text file with custom formatting
	//    - [x] (post-mvp) lookup digests and include them in
	// - [ ] (post-mvp) define index image schema and implement it
	// - [ ] (post-mvp) implement some usefull cheks
	//    - [ ] presence of Dockerfile.dockerignore in the same direcory
	// - [ ] (post-mvp) improve TagMode
	//    - [ ] expose various options for treating multiple tags
	//    - [ ] enable tags on release branches (either as an option
	//            or by documenting that upstream branch needs to
	//  		  change in repo config on a release branch)
	//    - [ ] enable semver tags along with tree hash tags by default
	//    - [ ] enable non-semver tags
	// - [ ] (post-mvp) should unnamed/main variants be allowed along with named variants?
	// - [ ] (post-mvp) export prefix
	// - [ ] (post-mvp) rewrite git package using a library

	ir := &recipe.ImagineRecipe{
		Push:      f.Push,
		Export:    f.Export,
		Platforms: f.Platforms,
		WorkDir:   initialWD,
		BuildSpec: &bc.Spec,
	}

	ir.Git.Git = repo
	ir.Git.BaseBranch = f.UpstreamBranch
	if !f.WithoutSuffix {
		ir.Git.BranchedOffSuffix = "-dev"    // TODO(post-mvp): make this a flag and a repo config field
		ir.Git.WorkInProgressSuffix = "-wip" // TODO(post-mvp): make this a repo and a repo config field
	}

	ir.Config.Data = bcData
	ir.Config.Path = bcPath

	m, err := ir.ToBakeManifest(f.Registries...)
	if err != nil {
		return err
	}

	rb := rebuilder.Rebuilder{
		RegistryAPI: &registry.Registry{},
	}

	fmt.Printf("current registry refs: %s\n", strings.Join(m.RegistryRefs(), ", "))

	rebuild, reason, err := rb.ShouldRebuild(m)
	if err != nil {
		return err
	}
	if f.Export {
		rebuild = true
		reason = "forcing image rebuild due to export option being set"
	}
	if f.Force {
		rebuild = true
		reason = "forcing image rebuild due to force option being set"
	}
	if !rebuild {
		fmt.Println("no need to rebuild")
		return nil
	}
	fmt.Println(reason)

	manifest, metadata, cleanup, err := ir.WriteManifest(stateDirPath, f.Registries...)
	if err != nil {
		return err
	}

	if f.Debug {
		fmt.Printf("writen manifest %v\n", manifest)
	}

	bx := buildx.New(stateDirPath)
	bx.Debug = f.Debug
	bx.Platforms = f.Platforms

	if err := bx.InitBuilder(f.Builder); err != nil {
		return err
	}

	bakeArgs := []string{"--metadata-file", metadata}
	if f.NoCache {
		bakeArgs = append(bakeArgs, "--no-cache")
	}

	if err := bx.Bake(manifest, bakeArgs...); err != nil {
		return err
	}
	md, err := buildx.LoadBakeMetadata(metadata)
	if err != nil {
		return err
	}

	summariser := md.ToBuildSummary(bc.Spec.Name)

	summaryOutput := os.Stdout
	if f.SummaryOutput != "-" {
		summaryOutput, err = os.Create(f.SummaryOutput)
		if err != nil {
			return err
		}
		defer summaryOutput.Close()
	}

	if f.SummaryFormat == "auto" {
		switch ext := filepath.Ext(f.SummaryOutput); ext {
		case ".json", ".yaml", ".yml":
			f.SummaryFormat = strings.TrimPrefix(ext, ".")
		default:
			f.SummaryFormat = "text"
		}
	}

	switch f.SummaryFormat {
	case "text":
		err = summariser.WriteText(summaryOutput)
	case "lines":
		err = summariser.WriteLines(summaryOutput)
	case "json":
		err = summariser.WriteJSON(summaryOutput)
	case "yaml", "yml":
		err = summariser.WriteYAML(summaryOutput)
	default:
		// TODO: check this early
		err = fmt.Errorf("unknown summary format %q", f.SummaryFormat)
	}
	if err != nil {
		return err
	}

	if !f.Debug {
		cleanup()
	} else {
		fmt.Printf("keeping %q and %q for debugging\n", manifest, metadata)
	}
	return nil
}
