package vercel_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vercel/terraform-provider-vercel/client"
)

func TestAcc_Project(t *testing.T) {
	testAccProject(t, "")
}

func TestAcc_ProjectWithTeamID(t *testing.T) {
	testAccProject(t, os.Getenv("VERCEL_TERRAFORM_TESTING_TEAM"))
}

func TestAcc_ProjectWithGitRepository(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test_git", ""),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfigWithGitRepo(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test_git", ""),
					resource.TestCheckResourceAttr("vercel_project.test_git", "git_repository.type", "github"),
					resource.TestCheckResourceAttr("vercel_project.test_git", "git_repository.repo", "vercel/next.js"),
				),
			},
		},
	})
}

func TestAcc_ProjectImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", ""),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", ""),
				),
			},
			{
				ResourceName:      "vercel_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccProjectExists(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		c := client.New(os.Getenv("VERCEL_API_TOKEN"))
		_, err := c.GetProject(context.TODO(), rs.Primary.ID, teamID)
		return err
	}
}

func testAccProjectDestroy(n, teamID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no projectID is set")
		}

		c := client.New(os.Getenv("VERCEL_API_TOKEN"))
		_, err := c.GetProject(context.TODO(), rs.Primary.ID, teamID)

		var apiErr client.APIError
		if err == nil {
			return fmt.Errorf("Found project but expected it to have been deleted")
		}
		if err != nil && errors.As(err, &apiErr) {
			if apiErr.StatusCode == 404 {
				return nil
			}
			return fmt.Errorf("Unexpected error checking for deleted project: %s", apiErr)
		}

		return err
	}
}

func testAccProject(t *testing.T, tid string) {
	extraConfig := ""
	testTeamID := resource.TestCheckNoResourceAttr("vercel_project.test", "team_id")
	if tid != "" {
		extraConfig = fmt.Sprintf(`team_id = "%s"`, tid)
		testTeamID = resource.TestCheckResourceAttr("vercel_project.test", "team_id", tid)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccProjectDestroy("vercel_project.test", tid),
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectConfig(extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccProjectExists("vercel_project.test", tid),
					testTeamID,
					resource.TestCheckResourceAttr("vercel_project.test", "name", "test-acc-project"),
					resource.TestCheckResourceAttr("vercel_project.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("vercel_project.test", "dev_command", "npm run serve"),
					resource.TestCheckResourceAttr("vercel_project.test", "framework", "create-react-app"),
					resource.TestCheckResourceAttr("vercel_project.test", "install_command", "npm install"),
					resource.TestCheckResourceAttr("vercel_project.test", "output_directory", ".output"),
					resource.TestCheckResourceAttr("vercel_project.test", "public_source", "true"),
					resource.TestCheckResourceAttr("vercel_project.test", "root_directory", "ui/src"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "foo",
						"value": "bar",
					}),
					resource.TestCheckTypeSetElemAttr("vercel_project.test", "environment.0.target.*", "production"),
				),
			},
			// Update testing
			{
				Config: testAccProjectConfigUpdated(extraConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vercel_project.test", "name", "test-acc-two"),
					resource.TestCheckNoResourceAttr("vercel_project.test", "build_command"),
					resource.TestCheckTypeSetElemNestedAttrs("vercel_project.test", "environment.*", map[string]string{
						"key":   "bar",
						"value": "baz",
					}),
				),
			},
		},
	})
}

func testAccProjectConfigUpdated(extras string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-two"
  %s
  environment = [
    {
      key    = "bar"
      value  = "baz"
      target = ["preview"]
    }
  ]
}
`, extras)
}

func testAccProjectConfigWithGitRepo() string {
	return `
resource "vercel_project" "test_git" {
  name = "test-acc-two"
  git_repository = {
    type = "github"
    repo = "vercel/next.js"
  }
}
    `
}

func testAccProjectConfig(extra string) string {
	return fmt.Sprintf(`
resource "vercel_project" "test" {
  name = "test-acc-project"
  %s
  build_command = "npm run build"
  dev_command = "npm run serve"
  framework = "create-react-app"
  install_command = "npm install"
  output_directory = ".output"
  public_source = true
  root_directory = "ui/src"
  environment = [
    {
      key    = "foo"
      value  = "bar"
      target = ["production"]
    }
  ]
}
`, extra)
}
