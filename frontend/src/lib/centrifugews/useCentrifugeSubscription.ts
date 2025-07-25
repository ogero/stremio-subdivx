import {useEffect} from "react";
import {useCentrifuge} from "./CentrifugeContext";

/**
 * Subscribe to Centrifuge channel in a React-friendly way.
 * Calls handler on every message.
 */
export function useCentrifugeSubscription(channel: string, handler: (msg: any) => void) {
    const {subscribe, unsubscribe} = useCentrifuge();

    useEffect(() => {
        subscribe(channel, handler);
        return () => {
            unsubscribe(channel, handler);
        };
    }, [channel, handler, subscribe, unsubscribe]);
}