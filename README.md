# go-schedule
This package provides a type-parameterized class `Scheduler[G]` which can be used to orchestrate sequences of synchronous and asynchronous steps. Specifically, it offers the following functionality and guarantees:
- `Scheduler[G]` can `RunSync()` (blocking), in which case all internal processing and user callbacks will be invoked on the calling goroutine; or `RunAsync()` (non-blocking), where one goroutine will be spawned for processing and callbacks.
- user code running on the processing goroutine can invoke `ProcessSync()` to synchronously execute an `Event[G]`, e.g. `ScheduleAsyncEvent[G]`, `ScheduleSequenceEvent[G]`, `ReleaseGroupSliceEvent[G]`. Otherwise, user code must invoke `ProcessAsync()` to enqueue an `Event[G]` for asynchronous execution.
- `ScheduleAsyncEvent[G]`:
  - an `AsyncVariant[G]` can be used to register arbitrary go `chan` to the scheduler. Upon receiving from such `chan`, the scheduler will invoke the user installed `SelectFunc` on the processing goroutine.
  - an `AsyncVariant[G]` can have arbitrary number of `groups` (type-parameterized `[G]`) associated. At runtime, `ReleaseGroupSliceEvent[G]` can be used to remove from the scheduler all registered `AsyncVariant[G]`s that match the given `group` criteria.  
    Note: from the processing goroutine perspective, once an `AsyncVariant[G]` is removed, it is guaranteed that the corresponding `chan` will not be received or selected by the scheduler on any subsequent process loop pass, even if the `chan` had been pushed data in the interim. This semantic is important for eliminating race condition between concurrent `chan` removal and `chan` reception.
  - upon remove and release of an `AsyncVariant[G]`, its user installed `ReleaseFunc` will be invoked, regardless of whether or how many times the corresponding `chan` has been selected.
  - for one-shot timer and recurring ticker variants, it is advised to use the `TimerAsync[G]()` and `TickerAsync[G]()` factory methods for simplified construction of `AsyncVariant[G]`.
- `ScheduleSequenceEvent[G]`:
  - a `Sequence[G]` can be consisted of zero or more `Step[G]`s. `Step[G]`s within a `Sequence[G]` will be executed in a serialized fashion, that is, a `Step[G]` cannot begin execution until all prior `Step[G]`s in the same containing `Sequence[G]` have completed.
  - a `Step[G]` can be one of the following:
    - an `ActionStep[G]`, which will execute the user installed `Action` functor immediately at its turn, synchronously on the processing goroutine. If the `Action` functor returns `nil`, the `ActionStep[G]` is deemed successful, and control will proceed synchronously to the next `Step[G]`; otherwise, the `ActionStep[G]` is deemed failed, and the containing `Sequence[G]` will be aborted.
    - a `TimerStep[G]`, which will schedule an `AsyncVariant[G]`, effectively pausing the containing `Sequence[G]` at this step for a given `Delay`, as specified by the user. If the underlying `AsyncVariant[G]`'s corresponding timer `chan` fires and gets selected, the `TimerStep[G]` is deemed successful, and control will proceed synchronously to the next `Step[G]`; if the underlying `AsyncVariant[G]` is released before being selected, the `TimerStep[G]` is deemed failed, and the containing `Sequence[G]` will be aborted.  
    Note: after scheduling of `AsyncVariant[G]` in a `TimerStep[G]`, control will be yielded on the processing goroutine, allowing `Event[G]`s or other `Sequence[G]`s to execute on the next pass.
    - a `SequenceStep[G]`, which can be used to nest a child `Sequence[G]` as a `Step[G]` within the parent `Sequence[G]`. Furthermore, user can specify `RepTotal` to control how many times the child `Sequence[G]` should execute. The `Step[G]` is considered completed only when all repetitions of the child `Sequence[G]` have completed. If any `Step[G]` in the child `Sequence[G]` fails, the failure will propagate upwards, unwinding the nesting chain and causing all `Sequence[G]`s to abort.
  - a `Sequence[G]` can have arbitrary number of `groups` (type-parameterized `[G]`) associated. For `TimerStep[G]`s in the `Sequence[G]`, these `groups` will be passed on to the underlying `AsyncVariant[G]`s. Subsequently, a `ReleaseGroupSliceEvent[G]` to the scheduler can be used to interrupt `TimerStep[G]`s with matching `group` criteria. In addition, `InheritGroup` can be set to `true` to enable automatic inheritance of `groups` from the parent `Sequence[G]` one level above, if any. This can be useful for scenarios where releasing a parent `Sequence[G]` `group` needs to affect child `Sequence[G]` `TimerStep[G]`s.
  - user can choose to install `StepResultFunc` and/or `SequenceResultFunc` functors on a `Sequence[G]`.
    - `StepResultFunc` will be invoked if any rep of the `Step[G]` fails, or if all reps of the `Step[G]` have completed.
    - `SequenceResultFunc` will be invoked if any `Step[G]` of the `Sequence[G]` fails, or if all `Step[G]`s of the `Sequence[G]` have completed.
- user can invoke `Shutdown()` to stop the scheduler and release relevant resources. This will cause all currently registered `AsyncVariant[G]`s to be removed and associated functors invoked serially.  
  Note: `Shutdown()` will block until `processLoop()` terminates.

Limits:
- the scheduler module uses `reflect.Select()` for dynamic selection of registered `chan`s. As such, the total number of outstanding `AsyncVariant[G]`s, whether registered directly via `ScheduleAsyncEvent[G]` or indirectly through `TimerStep[G]` via `ScheduleSequenceEvent[G]`, must not exceed 65534.  
  Note: one select case is reserved internally for `Event[G]` processing.
- the total number of `Step[G]` in one level of `Sequence[G]` must not exceed 65535.
- the total level of nesting of `Sequence[G]` must not exceed 65535.

Dependencies:
- [emirpasic/gods](https://github.com/emirpasic/gods) - for redblacktree
- [Meander-Cloud/go-chdyn](https://github.com/Meander-Cloud/go-chdyn) - for `chan` bridging
