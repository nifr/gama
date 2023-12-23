package usecase

import (
	"context"

	gr "github.com/termkit/gama/internal/github/repository"
	pw "github.com/termkit/gama/pkg/workflow"
	py "github.com/termkit/gama/pkg/yaml"
)

type useCase struct {
	githubRepository gr.Repository
}

func New(githubRepository gr.Repository) UseCase {
	return &useCase{
		githubRepository: githubRepository,
	}
}

func (u useCase) ListRepositories(ctx context.Context, input ListRepositoriesInput) (*ListRepositoriesOutput, error) {
	repositories, err := u.githubRepository.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}

	// Create a buffered channel for jobs, results and errors
	jobs := make(chan gr.GithubRepository, len(repositories))
	results := make(chan GithubRepository, len(repositories))
	errors := make(chan error, len(repositories))

	// Start a number of workers
	for w := 1; w <= len(repositories); w++ {
		go u.workerListRepositories(ctx, jobs, results, errors)
	}

	// Send jobs to the workers
	for _, repository := range repositories {
		jobs <- repository
	}
	close(jobs)

	// Collect the results and errors
	var result []GithubRepository
	var resultErr error
	for range repositories {
		select {
		case res := <-results:
			result = append(result, res)
		case err := <-errors:
			resultErr = err
		}
	}

	return &ListRepositoriesOutput{
		Repositories: result,
	}, resultErr
}

func (u useCase) workerListRepositories(ctx context.Context, jobs <-chan gr.GithubRepository, results chan<- GithubRepository, errors chan<- error) {
	for repository := range jobs {
		triggerableWorkflows, err := u.githubRepository.GetTriggerableWorkflows(ctx, repository.FullName)
		if err != nil {
			errors <- err
			continue
		}

		var workflows []Workflow
		for _, workflow := range triggerableWorkflows {
			workflows = append(workflows, Workflow{
				ID:    workflow.Id,
				Name:  workflow.Name,
				State: workflow.State,
			})
		}

		results <- GithubRepository{
			Name:                 repository.FullName,
			Stars:                repository.StargazersCount,
			Private:              repository.Private,
			DefaultBranch:        repository.DefaultBranch,
			TriggerableWorkflows: workflows,
		}
	}
}

func (u useCase) ListWorkflowByRepository(ctx context.Context, input ListWorkflowByRepositoryInput) (*ListWorkflowByRepositoryOutput, error) {
	//TODO implement me
	panic("implement me")
}

func (u useCase) ListWorkflowRuns(ctx context.Context, input ListWorkflowRunsInput) (*ListWorkflowRunsOutput, error) {
	//TODO implement me
	panic("implement me")
}

func (u useCase) InspectWorkflow(ctx context.Context, input InspectWorkflowInput) (*InspectWorkflowOutput, error) {
	workflowData, err := u.githubRepository.InspectWorkflowContent(ctx, input.Repository, input.WorkflowFile)
	if err != nil {
		return nil, err
	}

	workflowContent, err := py.UnmarshalWorkflowContent(workflowData)
	if err != nil {
		return nil, err
	}

	workflow, err := pw.ParseWorkflow(*workflowContent)
	if err != nil {
		return nil, err
	}

	return &InspectWorkflowOutput{
		Workflow: *workflow,
	}, nil
}

func (u useCase) TriggerWorkflow(ctx context.Context, input TriggerWorkflowInput) (*TriggerWorkflowOutput, error) {
	//TODO implement me
	panic("implement me")
}