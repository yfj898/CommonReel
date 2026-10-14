# Observed browser verification — 2026-10-13

Local browser at http://127.0.0.1:8092/ against the built Go server and real
MovieLens 1M data. These observations are separate from automated tests.

- Initial dataset provenance showed 1,000,209 research ratings.
- Three-person example created Alex/Jamie/Sam with fixed-seed training profile
  IDs 5222/4674/5179. Names and group membership are fictional.
- Group-average first pick was The Godfather; balanced first pick was
  Dr. Strangelove. Their displayed minimum predictions were approximately
  4.48 and 4.55. This demonstrates the objective, not human preference truth.
- Guided “no horror” added an owned veto through set_member. A second tab
  received the revision-5 update over SSE without a manual reload.
- Guided “why first” returned per-person reasons and the group tradeoff.
- Clicking choose saved the film with source revision 5; a browser reload
  restored the receipt from SQLite.
- Editing Sam to veto War produced revision 7, changed the first pick to
  The Godfather, and every displayed candidate lacked War and Horror.
- The old saved choice is an immutable historical receipt. The UI now flags
  when room preferences have changed since it was saved.
- Runtime console inspection returned no error or warning entries.
- Desktop layout visually inspected at the browser's default viewport. No
  mobile breakpoint or real Alexa device test was performed.

The screenshot commonreel-desktop.png is an actual local viewport capture,
not a design mockup or generated rendering. A video still needs recording.
