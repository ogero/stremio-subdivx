import {Coffee, Download, Star} from "lucide-react";
import {Link} from 'react-router-dom';


const Hero = () => {
    return (
        <section className="min-h-screen flex items-center justify-center bg-gradient-to-br from-black via-gray-900 to-purple-900 relative overflow-hidden">
            {/* Animated background elements */}
            <div className="absolute inset-0 opacity-20">
                <div className="absolute top-1/4 left-1/4 w-64 h-64 bg-purple-500 rounded-full blur-3xl animate-pulse"></div>
                <div className="absolute bottom-1/4 right-1/4 w-64 h-64 bg-pink-500 rounded-full blur-3xl animate-pulse delay-1000"></div>
            </div>

            <div className="container mx-auto px-4 py-20 text-center relative z-10">
                <div className="max-w-4xl mx-auto">
                    <div className="mb-8 animate-fade-in">
                        <div className="inline-flex items-center space-x-2 bg-purple-500/20 text-purple-300 px-4 py-2 rounded-full border border-purple-500/30 mb-6">
                            <Star size={16} className="fill-current"/>
                            <span className="text-sm font-medium">The definitive Spanish subtitles</span>
                        </div>
                    </div>

                    <h1 className="text-5xl md:text-7xl font-bold text-white mb-6 leading-tight animate-fade-in">
                        Never Miss
                        <span className="bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent block">A Single Line</span>
                    </h1>

                    <p className="text-xl md:text-2xl text-gray-300 mb-8 max-w-3xl mx-auto leading-relaxed animate-fade-in">
                        Access the ultimate library of Spanish subtitles from Subdivx right inside Stremio. Enjoy your favorite movies and series with perfect timing.
                    </p>

                    <div className="flex flex-col sm:flex-row items-center justify-center space-y-4 sm:space-y-0 sm:space-x-6 animate-fade-in">
                        <Link to='stremio://stremio-subdivx.xor.ar/manifest.json'>
                            <button
                                className="bg-gradient-to-r from-purple-500 to-pink-500 text-white px-8 py-4 rounded-xl font-semibold text-lg hover:shadow-lg hover:shadow-purple-500/25 transition-all duration-300 hover:scale-105 flex items-center space-x-2">
                                <Download size={20}/>
                                <span>Install Now</span>
                            </button>
                        </Link>

                        <Link to={'https://cafecito.app/ogero'} rel={'noopener'} target={'_blank'}>
                            <button className="border border-gray-600 text-white px-8 py-4 rounded-xl font-semibold text-lg hover:bg-white/10 transition-all duration-300 flex items-center space-x-2">
                                <Coffee size={20}/>
                                <span>Invitame un café en cafecito.app</span>
                            </button>
                        </Link>
                    </div>

                    <div className="mt-12 p-6 bg-black/30 backdrop-blur-sm rounded-xl border border-gray-700">
                        <p className="text-gray-300 mb-4">Or manually install it by copying the manifest URL:</p>
                        <div className="bg-gray-900 p-4 rounded-lg">
                            <code className="text-purple-400 font-mono">https://stremio-subdivx.xor.ar/manifest.json</code>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    );
};

export default Hero;
