---------------------------- MODULE EventCounter ----------------------------

EXTENDS Integers

\* The number of ``events'' that have occurred (always 0 if we're not keeping track).
VARIABLE nEvents

\* The maximum number of events to allow, or ``-1'' for unlimited.
maxEvents == -1

InitEvents ==
  nEvents = 0       \* Start with the counter at zero

(* If we're counting events, increment the event counter.
   We don't increment the counter when we don't have a maximum because that
   would make the model infinite.
   Actions tagged with CountEvent cannot happen once nEvents = maxEvents. *)
CountEvent ==
  IF maxEvents = -1 THEN
    UNCHANGED nEvents
  ELSE
    /\ nEvents < maxEvents
    /\ nEvents' = nEvents + 1

=============================================================================