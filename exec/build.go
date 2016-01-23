package exec

import (
	"database/sql/driver"
	"fmt"
	"strings"

	u "github.com/araddon/gou"

	"github.com/araddon/qlbridge/expr"
	"github.com/araddon/qlbridge/plan"
	"github.com/araddon/qlbridge/rel"
)

var (
	// Standard errors
	ErrShuttingDown     = fmt.Errorf("Received Shutdown Signal")
	ErrNotSupported     = fmt.Errorf("QLBridge: Not supported")
	ErrNotImplemented   = fmt.Errorf("QLBridge: Not implemented")
	ErrUnknownCommand   = fmt.Errorf("QLBridge: Unknown Command")
	ErrInternalError    = fmt.Errorf("QLBridge: Internal Error")
	ErrNoSchemaSelected = fmt.Errorf("No Schema Selected")

	_ = u.EMPTY

	// JobBuilder implements JobRunner
	_ JobRunner = (*JobBuilder)(nil)
	// Ensure that we implement the expr.Visitor interface
	_ rel.Visitor       = (*JobBuilder)(nil)
	_ rel.SourceVisitor = (*SourceBuilder)(nil)
)

// Job Runner is the main RunTime interface for running a SQL Job
type JobRunner interface {
	Setup() error
	Run() error
	Close() error
}

// SqlJob is dag of tasks for sql execution
// This is a simple Job Builder
//   hopefully we create smarter ones but this is a basic implementation for
///  running in-process, not distributed
type JobBuilder struct {
	Visitor   rel.Visitor
	RootTask  TaskRunner
	Ctx       *plan.Context
	TaskMaker plan.TaskPlanner
	distinct  bool
	children  []plan.Task
	//where     expr.Node
}
type SourceBuilder struct {
	SourceVisitor rel.SourceVisitor
	Plan          *plan.SourcePlan
	TaskMaker     plan.TaskPlanner
}

func NewJobBuilder(ctx *plan.Context, visitor rel.Visitor) *JobBuilder {
	b := &JobBuilder{}
	b.Ctx = ctx
	if visitor == nil {
		b.Visitor = b
	} else {
		b.Visitor = visitor
	}
	b.TaskMaker = TaskRunnersMaker(ctx)
	return b
}
func BuildSqlJob(ctx *plan.Context) (*JobBuilder, error) {
	job := NewJobBuilder(ctx, nil)
	task, err := BuildSqlJobVisitor(job, ctx)
	taskRunner, ok := task.(TaskRunner)
	if !ok {
		return nil, fmt.Errorf("Expected TaskRunner but was %T", task)
	}
	job.RootTask = taskRunner
	return job, err
}

// Create Job made up of sub-tasks in DAG that is the
//  plan for execution of this query/job
func BuildSqlJobVisitor(visitor rel.Visitor, ctx *plan.Context) (rel.Task, error) {

	stmt, err := rel.ParseSql(ctx.Raw)
	if err != nil {
		u.Debugf("could not parse %v", err)
		return nil, err
	}
	if stmt == nil {
		return nil, fmt.Errorf("Not statement for parse? %v", ctx.Raw)
	}
	ctx.Stmt = stmt

	if ctx.Schema == nil {
		u.LogTraceDf(u.WARN, 12, "no schema? %s", ctx.Raw)
	}

	u.Debugf("build sqljob.Visitor: %T   %#v", visitor, visitor)
	task, _, err := stmt.Accept(visitor)
	//u.Debugf("build sqljob.proj: %p", builder.Projection)

	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("No task found? %v", ctx.Raw)
	}
	return task, err
}

func NewSourceBuilder(sp *plan.SourcePlan, taskMaker plan.TaskPlanner) *SourceBuilder {
	return &SourceBuilder{Plan: sp, TaskMaker: taskMaker}
}

// func (m *JobBuilder) Wrap(visitor rel.Visitor) rel.Visitor {
// 	u.Debugf("wrap %T", visitor)
// 	m.Visitor = visitor
// 	return m
// }

func (m *JobBuilder) VisitInto(stmt *rel.SqlInto) (rel.Task, rel.VisitStatus, error) {
	u.Debugf("VisitInto %+v", stmt)
	return nil, rel.VisitError, expr.ErrNotImplemented
}

func (m *JobBuilder) VisitPreparedStmt(stmt *rel.PreparedStatement) (rel.Task, rel.VisitStatus, error) {
	u.Debugf("VisitPreparedStmt %+v", stmt)
	return nil, rel.VisitError, expr.ErrNotImplemented
}

func (m *JobBuilder) VisitCommand(stmt *rel.SqlCommand) (rel.Task, rel.VisitStatus, error) {
	u.Debugf("VisitCommand %+v", stmt)
	return nil, rel.VisitError, expr.ErrNotImplemented
}

func (m *JobBuilder) Setup() error {
	if m == nil {
		return fmt.Errorf("No job")
	}
	if m.RootTask == nil {
		return fmt.Errorf("No task exists for this job")
	}
	return m.RootTask.Setup(0)
}

func (m *JobBuilder) Run() error {
	if m.Ctx != nil {
		m.Ctx.DisableRecover = m.Ctx.DisableRecover
	}
	return m.RootTask.Run()
}

func (m *JobBuilder) Close() error {
	return m.RootTask.Close()
}

// The drain is the last out channel, on last task
func (m *JobBuilder) DrainChan() MessageChan {
	tasks := m.RootTask.Children()
	return tasks[len(tasks)-1].(TaskRunner).MessageOut()
}

// Create a multiple error type
type errList []error

func (e *errList) append(err error) {
	if err != nil {
		*e = append(*e, err)
	}
}

func (e errList) error() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

func (e errList) Error() string {
	a := make([]string, len(e))
	for i, v := range e {
		a[i] = v.Error()
	}
	return strings.Join(a, "\n")
}

func params(args []driver.Value) []interface{} {
	r := make([]interface{}, len(args))
	for i, v := range args {
		r[i] = interface{}(v)
	}
	return r
}
