package rel

// VisitStatus surfaces status to visit builders
// if visit was completed, successful or needs to be polyfilled
type VisitStatus int

const (
	VisitUnknown  VisitStatus = 0 // not used
	VisitError    VisitStatus = 1 // error
	VisitFinal    VisitStatus = 2 // final, no more building needed
	VisitContinue VisitStatus = 3 // continue visit
)

// Visitor defines the Visit Pattern, so our expr package can
//   expect implementations from downstream packages
//   in our case: planner(s), job builder, execution engine
//
type Visitor interface {
	// Wrap allows a Visitor to have an implementation higher up the stack
	//  "Override" behavior
	//Wrap(Visitor) Visitor
	VisitPreparedStmt(stmt *PreparedStatement) (Task, VisitStatus, error)
	VisitSelect(stmt *SqlSelect) (Task, VisitStatus, error)
	VisitInsert(stmt *SqlInsert) (Task, VisitStatus, error)
	VisitUpsert(stmt *SqlUpsert) (Task, VisitStatus, error)
	VisitUpdate(stmt *SqlUpdate) (Task, VisitStatus, error)
	VisitDelete(stmt *SqlDelete) (Task, VisitStatus, error)
	VisitShow(stmt *SqlShow) (Task, VisitStatus, error)
	VisitDescribe(stmt *SqlDescribe) (Task, VisitStatus, error)
	VisitCommand(stmt *SqlCommand) (Task, VisitStatus, error)
	VisitInto(stmt *SqlInto) (Task, VisitStatus, error)

	// Select Components
	VisitWhere(stmt *SqlSelect) (Task, VisitStatus, error)
	VisitHaving(stmt *SqlSelect) (Task, VisitStatus, error)
	VisitGroupBy(stmt *SqlSelect) (Task, VisitStatus, error)
	VisitProjection(stmt *SqlSelect) (Task, VisitStatus, error)
	//VisitMutateWhere(stmt *SqlWhere) (Task, VisitStatus, error)
}

// Interface for sub-select Tasks of the Select Statement
type SourceVisitor interface {
	VisitSourceSelect() (Task, VisitStatus, error)
	VisitSource(scanner interface{} /*schema.Scanner*/) (Task, VisitStatus, error)
	VisitSourceJoin(scanner interface{} /*schema.Scanner*/) (Task, VisitStatus, error)
	VisitWhere() (Task, VisitStatus, error)
}
