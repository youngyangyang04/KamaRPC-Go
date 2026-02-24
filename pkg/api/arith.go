package api

type Args struct {
	A int
	B int
}

type Args1 struct {
	A int
	B int
	C int
}

type Reply struct {
	Result int
}

type Arith struct {
}

func (a *Arith) Add(args *Args, reply *Reply) error {
	reply.Result = args.A + args.B
	// log.Println("Add result is ", reply.Result)
	return nil
}

func (a *Arith) Mul(args *Args, reply *Reply) error {
	reply.Result = args.A * args.B
	// log.Println("Add result is ", reply.Result)
	return nil
}

type Arith2 struct {
}

func (a *Arith2) Add(args *Args1, reply *Reply) error {
	reply.Result = args.A + args.B + args.C
	// log.Println("Add result is ", reply.Result)
	return nil
}

func (a *Arith2) Mul(args *Args1, reply *Reply) error {
	reply.Result = args.A * args.B * args.C
	// log.Println("Add result is ", reply.Result)
	return nil
}
