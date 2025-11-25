package kernel

func NewApplicationID(id string) ApplicationID { return ApplicationID(id) }
func (r ApplicationID) String() string         { return string(r) }
func (r ApplicationID) IsEmpty() bool          { return string(r) == "" }

type CandidateID string

func NewCandidateID(id string) CandidateID { return CandidateID(id) }
func (r CandidateID) String() string       { return string(r) }
func (r CandidateID) IsEmpty() bool        { return string(r) == "" }

type JobID string

func NewJobID(id string) JobID { return JobID(id) }
func (r JobID) String() string { return string(r) }
func (r JobID) IsEmpty() bool  { return string(r) == "" }
