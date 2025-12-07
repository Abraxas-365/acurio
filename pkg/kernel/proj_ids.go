package kernel

type CandidateID string

func NewCandidateID(id string) CandidateID { return CandidateID(id) }
func (r CandidateID) String() string       { return string(r) }
func (r CandidateID) IsEmpty() bool        { return string(r) == "" }

type JobID string

func NewJobID(id string) JobID { return JobID(id) }
func (r JobID) String() string { return string(r) }
func (r JobID) IsEmpty() bool  { return string(r) == "" }

type ResumeID string

func NewResumeID(id string) ResumeID { return ResumeID(id) }
func (r ResumeID) String() string    { return string(r) }
func (r ResumeID) IsEmpty() bool     { return string(r) == "" }
