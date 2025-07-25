import {Download, Play, Search} from "lucide-react";
import {useTranslation} from "react-i18next";
import {useCallback, useState} from "react";
import {useCentrifugeSubscription} from "@/lib/centrifugews/useCentrifugeSubscription.ts";
import {Skeleton} from "@/components/ui/skeleton.tsx";

const STATS_CHANNEL = "stremio-subdivx:stats";

type StatsResponse = {
  searchesCount24: number;
  downloadsCount24: number;
  titleInstant: string;
}

function isStatsResponse(data: unknown): data is StatsResponse {
  return (
    typeof data === 'object' &&
    data !== null &&
    'searchesCount24' in data &&
    'downloadsCount24' in data &&
    'titleInstant' in data &&
    typeof (data as StatsResponse).searchesCount24 === 'number' &&
    typeof (data as StatsResponse).downloadsCount24 === 'number' &&
    typeof (data as StatsResponse).titleInstant === 'string'
  );
}


const Stats = () => {

  const {t} = useTranslation();
  const [data, setData] = useState<StatsResponse | null>(null);

  const handleStats = useCallback((data: unknown) => {
    try {
      // Add runtime type checking
      if (isStatsResponse(data)) {
        setData(data);
      } else {
        console.error('Invalid stats data received:', data);
      }
    } catch (error) {
      console.error('Error processing stats:', error);
    }
  }, []);

  useCentrifugeSubscription(STATS_CHANNEL, handleStats);

  return (
    <section id="how-it-works" className="py-20 bg-black">
      <div className="container mx-auto px-4">
        <div className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl font-bold text-white mb-6">
            {t("What are we")} <span className="bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">{t("watching")}</span>?
          </h2>
          <p className="text-xl text-gray-300 max-w-2xl mx-auto">
            {t("take a peek")}
          </p>
        </div>

        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">

            <div className="text-center relative">
              <div className="relative z-10">
                <div className="w-32 h-32 bg-gradient-to-r from-purple-500 to-pink-500 rounded-full flex items-center justify-center mx-auto mb-6 relative">
                  <div className="w-24 h-24 bg-black rounded-full flex items-center justify-center">
                    <Search size={32} className="text-white"/>
                  </div>
                </div>

                <h3 className="text-2xl font-bold text-white mb-4">{t("Searches")}</h3>
                {
                  data && data.searchesCount24 ?
                    (<p className="text-gray-300 leading-relaxed">{t("searches_last_24_hours", {count: data.searchesCount24})}</p>)
                    :
                    <div className={"flex justify-center"}><Skeleton className="h-4 w-[200px] cen"/></div>
                }
              </div>
            </div>

            <div className="text-center relative">
              <div className="relative z-10">
                <div className="w-32 h-32 bg-gradient-to-r from-purple-500 to-pink-500 rounded-full flex items-center justify-center mx-auto mb-6 relative">
                  <div className="w-24 h-24 bg-black rounded-full flex items-center justify-center">
                    <Download size={32} className="text-white"/>
                  </div>
                </div>

                <h3 className="text-2xl font-bold text-white mb-4">{t("Downloads")}</h3>
                {
                  data && data.downloadsCount24 ?
                    (<p className="text-gray-300 leading-relaxed">{t("downloads_last_24_hours", {count: data.downloadsCount24})}</p>)
                    :
                    <div className={"flex justify-center"}><Skeleton className="h-4 w-[200px] cen"/></div>
                }
              </div>
            </div>

            <div className="text-center relative">
              <div className="relative z-10">
                <div className="w-32 h-32 bg-gradient-to-r from-purple-500 to-pink-500 rounded-full flex items-center justify-center mx-auto mb-6 relative">
                  <div className="w-24 h-24 bg-black rounded-full flex items-center justify-center">
                    <Play size={32} className="text-white"/>
                  </div>
                </div>

                <h3 className="text-2xl font-bold text-white mb-4">{t("Last seen")}</h3>
                {
                  data && data.titleInstant ?
                    (<p className="text-gray-300 leading-relaxed">{t("last_seen_title", {title: data.titleInstant})}</p>)
                    :
                    <div className={"flex justify-center"}><Skeleton className="h-4 w-[200px] cen"/></div>
                }
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default Stats;