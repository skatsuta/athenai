package peco

func (l *Location) SetColumn(n int) {
	l.col = n
}

func (l Location) Column() int {
	return l.col
}

func (l *Location) SetLineNumber(n int) {
	l.lineno = n
}

func (l Location) LineNumber() int {
	return l.lineno
}

func (l *Location) SetOffset(n int) {
	l.offset = n
}

func (l Location) Offset() int {
	return l.offset
}

func (l *Location) SetPerPage(n int) {
	l.perPage = n
}

func (l Location) PerPage() int {
	return l.perPage
}

func (l *Location) SetPage(n int) {
	l.page = n
}

func (l Location) Page() int {
	return l.page
}

func (l *Location) SetTotal(n int) {
	l.total = n
}

func (l Location) Total() int {
	return l.total
}

func (l *Location) SetMaxPage(n int) {
	l.maxPage = n
}

func (l Location) MaxPage() int {
	return l.maxPage
}

func (l Location) PageCrop() PageCrop {
	return PageCrop{
		perPage:     l.perPage,
		currentPage: l.page,
	}
}

// Crop returns a new Buffer whose contents are
// bound within the given range
func (pf PageCrop) Crop(in Buffer) *FilteredBuffer {
	return NewFilteredBuffer(in, pf.currentPage, pf.perPage)
}
