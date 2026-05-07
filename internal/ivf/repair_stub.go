package ivf

func (idx *Index) repairIVF(_ [Dim]int16, _ *ivfSearchState) {}
func needsApprovalRepair(_ [Dim]int16) bool                   { return false }
func needsDenialRepair(_ [Dim]int16) bool                     { return false }
func needsKnownLateDenialRepair(_ [Dim]int16) bool            { return false }
