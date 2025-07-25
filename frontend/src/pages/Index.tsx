import Header from "@/components/Header";
import Hero from "@/components/Hero";
import Footer from "@/components/Footer.tsx";
import {CentrifugeProvider} from "@/lib/centrifugews/CentrifugeContext.tsx";
import Stats from "@/components/Stats.tsx";

const Index = () => {

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  const wsUrl = `${protocol}//${host}/ws`;

  return (
    <CentrifugeProvider url={wsUrl}>
      <div className="min-h-screen bg-black">
        <Header/>
        <Hero/>
        <Stats/>
        <Footer/>
      </div>
    </CentrifugeProvider>
  );
};

export default Index;
