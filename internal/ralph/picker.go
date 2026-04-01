package ralph

// Picker selects the next task to execute from a task source.
type Picker interface {
	// Pick returns the next task to execute, or nil if none are available.
	Pick() (*Task, error)
	// Update persists a status change for a task.
	Update(task Task) error
}

// FilePicker reads tasks from a JSON file on disk.
type FilePicker struct {
	Path string
}

// NewFilePicker creates a Picker backed by a task file.
func NewFilePicker(path string) *FilePicker {
	return &FilePicker{Path: path}
}

// Pick reads the task file and returns the highest-priority pending task.
func (fp *FilePicker) Pick() (*Task, error) {
	tasks, err := ParseTaskFile(fp.Path)
	if err != nil {
		return nil, err
	}
	return NextPending(tasks), nil
}

// Update reads the task file, updates the matching task, and writes it back.
func (fp *FilePicker) Update(task Task) error {
	tasks, err := ParseTaskFile(fp.Path)
	if err != nil {
		return err
	}
	for i, t := range tasks {
		if t.ID == task.ID {
			tasks[i] = task
			return WriteTaskFile(fp.Path, tasks)
		}
	}
	return nil
}
