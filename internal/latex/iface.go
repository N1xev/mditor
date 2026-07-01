package latex

// DeepEqCfg is unused by the renderer but is required for the Node interface
// in ast.go (which mathcha's debug.go satisfied with real implementations).
// Stubs here let the vendored types compile under Go 1.26's stricter
// interface satisfaction rules.
type DeepEqCfg struct {
	SkipPos bool
}

func (x *BadExpr) DeepEq(other Expr) bool          { _, ok := other.(*BadExpr); return ok }
func (x *BadExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*BadExpr)
	return ok
}
func (x *EmptyExpr) DeepEq(other Expr) bool          { _, ok := other.(*EmptyExpr); return ok }
func (x *EmptyExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*EmptyExpr)
	return ok
}
func (x *NumberLit) DeepEq(other Expr) bool          { _, ok := other.(*NumberLit); return ok }
func (x *NumberLit) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*NumberLit)
	return ok
}
func (x *VarLit) DeepEq(other Expr) bool          { _, ok := other.(*VarLit); return ok }
func (x *VarLit) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*VarLit)
	return ok
}
func (x *CompositeExpr) DeepEq(other Expr) bool          { _, ok := other.(*CompositeExpr); return ok }
func (x *CompositeExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*CompositeExpr)
	return ok
}
func (x *UnboundCompExpr) DeepEq(other Expr) bool          { _, ok := other.(*UnboundCompExpr); return ok }
func (x *UnboundCompExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*UnboundCompExpr)
	return ok
}
func (x *ParenCompExpr) DeepEq(other Expr) bool          { _, ok := other.(*ParenCompExpr); return ok }
func (x *ParenCompExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*ParenCompExpr)
	return ok
}
func (x *EnvExpr) DeepEq(other Expr) bool          { _, ok := other.(*EnvExpr); return ok }
func (x *EnvExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*EnvExpr)
	return ok
}
func (x *SimpleOpLit) DeepEq(other Expr) bool          { _, ok := other.(*SimpleOpLit); return ok }
func (x *SimpleOpLit) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*SimpleOpLit)
	return ok
}
func (x *UnknownCmdLit) DeepEq(other Expr) bool          { _, ok := other.(*UnknownCmdLit); return ok }
func (x *UnknownCmdLit) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*UnknownCmdLit)
	return ok
}
func (x RawRuneLit) DeepEq(other Expr) bool          { _, ok := other.(RawRuneLit); return ok }
func (x RawRuneLit) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(RawRuneLit)
	return ok
}
func (x *SimpleCmdLit) DeepEq(other Expr) bool          { _, ok := other.(*SimpleCmdLit); return ok }
func (x *SimpleCmdLit) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*SimpleCmdLit)
	return ok
}
func (x *SuperExpr) DeepEq(other Expr) bool          { _, ok := other.(*SuperExpr); return ok }
func (x *SuperExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*SuperExpr)
	return ok
}
func (x *SubExpr) DeepEq(other Expr) bool          { _, ok := other.(*SubExpr); return ok }
func (x *SubExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*SubExpr)
	return ok
}
func (x *Cmd1ArgExpr) DeepEq(other Expr) bool          { _, ok := other.(*Cmd1ArgExpr); return ok }
func (x *Cmd1ArgExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*Cmd1ArgExpr)
	return ok
}
func (x *Cmd2ArgExpr) DeepEq(other Expr) bool          { _, ok := other.(*Cmd2ArgExpr); return ok }
func (x *Cmd2ArgExpr) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*Cmd2ArgExpr)
	return ok
}
func (x *TextContainer) DeepEq(other Expr) bool          { _, ok := other.(*TextContainer); return ok }
func (x *TextContainer) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*TextContainer)
	return ok
}
func (x *TextStringWrapper) DeepEq(other Expr) bool          { _, ok := other.(*TextStringWrapper); return ok }
func (x *TextStringWrapper) DeepEqWith(other Expr, _ DeepEqCfg) bool {
	_, ok := other.(*TextStringWrapper)
	return ok
}
