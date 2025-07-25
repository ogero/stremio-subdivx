import React, {useState} from "react";
import {useCentrifuge} from "./CentrifugeContext";

export const SendMessageComponent: React.FC<{ channel: string }> = ({channel}) => {
  const {publish} = useCentrifuge();
  const [message, setMessage] = useState("");

  const sendMessage = async () => {
    if (message.trim()) {
      await publish(channel, {text: message});
      setMessage("");
    }
  };

  return (
    <div>
      <input
        value={message}
        onChange={e => setMessage(e.target.value)}
        placeholder="Send message"
      />
      <button onClick={sendMessage}>Send</button>
    </div>
  );
};