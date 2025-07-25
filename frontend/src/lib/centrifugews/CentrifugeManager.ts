import {Centrifuge, Subscription} from "centrifuge";

type MessageHandler = (message: any) => void;

type HandlerSet = Set<MessageHandler>;

class CentrifugeManager {
    private static instance: CentrifugeManager | null = null;
    private centrifuge: Centrifuge;
    private pubsubHandlers: Map<string, HandlerSet> = new Map();
    private connected: boolean = false;

    private constructor(url: string, options?: object) {
        this.centrifuge = new Centrifuge(url, options);

        this.centrifuge.on("connected", () => {
            this.connected = true;
        });

        this.centrifuge.on("disconnected", () => {
            this.connected = false;
        });

        this.centrifuge.connect();
    }

    static getInstance(url: string, options?: object) {
        if (!CentrifugeManager.instance) {
            CentrifugeManager.instance = new CentrifugeManager(url, options);
        }
        return CentrifugeManager.instance;
    }

    subscribe(channel: string, handler: MessageHandler): Subscription {

        if (!this.pubsubHandlers.has(channel)) {
            this.pubsubHandlers.set(channel, new Set<MessageHandler>());
        }

        const handlerSet = this.pubsubHandlers.get(channel)!;
        handlerSet.add(handler);

        const sub = this.centrifuge.getSubscription(channel);
        if (sub === null) {
            try {
                const sub = this.centrifuge.newSubscription(channel);

                sub.on("publication", (ctx) => {
                    const handlers = this.pubsubHandlers.get(channel);
                    if (handlers) {
                        handlers.forEach(h => {
                            try {
                                h(ctx.data);
                            } catch (err) {
                                console.error("Centrifuge handler error", err);
                            }
                        });
                    }
                });

                sub.subscribe();
                return sub;
            } catch (err) {
                console.error("Error subscribing to channel:", err);
            }
        }

        return sub;
    }

    unsubscribe(channel: string, handler: MessageHandler) {
        console.log("CentrifugeManager.unsubscribe", channel, handler);
        const handlerSet = this.pubsubHandlers.get(channel);

        if (handlerSet) {
            handlerSet.delete(handler);

            if (handlerSet.size === 0) {
                const sub = this.centrifuge.getSubscription(channel);
                if (sub) {
                    sub.unsubscribe();
                    sub.removeAllListeners();
                }
                this.pubsubHandlers.delete(channel);
            }
        }
    }

    publish(channel: string, data: any) {
        return this.centrifuge.publish(channel, data);
    }

    getConnectionState() {
        return this.connected ? "connected" : "disconnected";
    }

    disconnect() {
        this.centrifuge.disconnect();
        CentrifugeManager.instance = null;
    }
}

export default CentrifugeManager;